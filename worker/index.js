// Cloudflare Worker entry point. Wraps the Go-compiled WASM module exposed
// at globalThis.voidsliceLint by cmd/voidslice-wasm/main.go and serves the
// same /health + /lint API as internal/server.
//
// The .wasm file is bundled at deploy time via wrangler's default
// CompiledWasm rule for *.wasm imports — the import below resolves to a
// WebAssembly.Module. ES module workers cannot use the [wasm_modules]
// config form.

import "./wasm_exec.js";
import voidsliceWasm from "./voidslice.wasm";

const MAX_BODY_BYTES = 1 << 20; // 1 MiB
const ALLOW_METHODS = "POST, GET, OPTIONS";
const ALLOW_HEADERS = "Content-Type";
const LINT_CACHE_TTL_SECONDS = 300;
const LINT_CACHE_HEADER = "X-Voidslice-Cache";

let wasmReady = null;

function ensureWasm() {
  if (wasmReady) return wasmReady;
  wasmReady = (async () => {
    // eslint-disable-next-line no-undef
    const go = new Go();
    const instance = await WebAssembly.instantiate(voidsliceWasm, go.importObject);
    // go.run() never resolves (the Go program calls select{}).
    // Don't await it — let it run in the background.
    go.run(instance);
    // The Go runtime registers globalThis.voidsliceLint inside its main().
    // One microtask tick is enough for that registration to settle.
    await new Promise((resolve) => setTimeout(resolve, 0));
    if (typeof globalThis.voidsliceLint !== "function") {
      throw new Error("voidsliceLint export missing — WASM build is wrong");
    }
  })();
  return wasmReady;
}

function parseList(s) {
  return (s || "")
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);
}

// Origin is allowed if it exactly matches an entry in ALLOWED_ORIGINS, or if
// it parses as an https URL whose hostname ends with one of the leading-dot
// suffixes in ALLOWED_ORIGIN_SUFFIXES. The leading-dot convention is what
// enforces a label boundary — without it, ".voidslice.pages.dev" would also
// match "evilvoidslice.pages.dev".
function isAllowedOrigin(origin, env) {
  if (!origin) return false;
  if (parseList(env.ALLOWED_ORIGINS).includes(origin)) return true;
  const suffixes = parseList(env.ALLOWED_ORIGIN_SUFFIXES);
  if (suffixes.length === 0) return false;
  let url;
  try {
    url = new URL(origin);
  } catch {
    return false;
  }
  if (url.protocol !== "https:") return false;
  const host = url.hostname.toLowerCase();
  for (const sfx of suffixes) {
    const s = sfx.toLowerCase();
    if (s.startsWith(".") && host.endsWith(s)) return true;
  }
  return false;
}

// Per-request CORS headers. If the request carries an allowed Origin we echo
// it; otherwise we emit only Vary: Origin so caches differentiate without
// leaking permissive ACAO. Callers that need a Vary baseline (404, 405) get
// it for free this way.
function corsHeaders(req, env) {
  const origin = req.headers.get("Origin");
  if (origin && isAllowedOrigin(origin, env)) {
    return {
      "Access-Control-Allow-Origin": origin,
      "Access-Control-Allow-Methods": ALLOW_METHODS,
      "Access-Control-Allow-Headers": ALLOW_HEADERS,
      Vary: "Origin",
    };
  }
  return { Vary: "Origin" };
}

function jsonResponse(status, body, req, env, extraHeaders) {
  return new Response(typeof body === "string" ? body : JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
      ...corsHeaders(req, env),
      ...(extraHeaders || {}),
    },
  });
}

function textResponse(status, body, req, env) {
  return new Response(body, {
    status,
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      ...corsHeaders(req, env),
    },
  });
}

// /health stays public — no allowlist gate, no Vary, ACAO: * so curl-based
// monitors and uptime checkers from any origin work without configuration.
function healthResponse() {
  return new Response(JSON.stringify({ status: "ok" }), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
  });
}

function isAllowedContentType(ct) {
  if (!ct) return true;
  const head = ct.split(";")[0].trim().toLowerCase();
  if (head === "application/octet-stream") return true;
  if (head.startsWith("text/")) return true;
  if (head.startsWith("multipart/form-data")) return true;
  if (
    head.startsWith("image/") ||
    head.startsWith("audio/") ||
    head.startsWith("video/")
  ) {
    return false;
  }
  switch (head) {
    case "application/zip":
    case "application/gzip":
    case "application/x-gzip":
    case "application/x-tar":
    case "application/pdf":
      return false;
  }
  return true;
}

async function readWithCap(req) {
  const buf = await req.arrayBuffer();
  if (buf.byteLength > MAX_BODY_BYTES) {
    const err = new Error("body too large");
    err.statusCode = 413;
    throw err;
  }
  return buf;
}

// Version stamp folded into every /lint cache key. CF_VERSION_METADATA is the
// canonical Cloudflare binding; WORKER_VERSION is a plain-string fallback for
// dev/harness. "dev" is the last-resort default — local `wrangler dev` without
// a configured binding still gets a stable key, just one that doesn't roll
// over on edits.
function workerVersion(env) {
  return env?.CF_VERSION_METADATA?.id || env?.WORKER_VERSION || "dev";
}

function u32be(n) {
  return new Uint8Array([(n >>> 24) & 0xff, (n >>> 16) & 0xff, (n >>> 8) & 0xff, n & 0xff]);
}

// SHA-256 of length-prefixed (version, filename, body-bytes). Length prefixes
// prevent collisions across (filename="a", body="bc") vs (filename="ab",
// body="c"). Hashing raw body bytes — rather than the decoded string — lets
// the hit path skip the TextDecoder entirely.
async function lintCacheKey(req, env, filename, bodyBytes) {
  const enc = new TextEncoder();
  const v = enc.encode(workerVersion(env));
  const f = enc.encode(filename);
  const total = new Uint8Array(4 + v.length + 4 + f.length + 4 + bodyBytes.byteLength);
  let off = 0;
  total.set(u32be(v.length), off);
  off += 4;
  total.set(v, off);
  off += v.length;
  total.set(u32be(f.length), off);
  off += 4;
  total.set(f, off);
  off += f.length;
  total.set(u32be(bodyBytes.byteLength), off);
  off += 4;
  total.set(new Uint8Array(bodyBytes), off);
  const digest = await crypto.subtle.digest("SHA-256", total);
  const hex = Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, "0")).join("");
  const origin = new URL(req.url).origin;
  return new Request(`${origin}/__cache/lint/${hex}`, { method: "GET" });
}

async function handleLint(req, env) {
  // Origin gate runs before the method check, the rate limiter, and any
  // body work. A foreign-origin "simple" POST (e.g. Content-Type:
  // text/plain) skips preflight, so without this short-circuit the linter
  // would still run and burn the per-IP rate-limit budget even though the
  // browser drops the response. Requests with no Origin (curl,
  // server-to-server) bypass the gate — there's no CORS context to enforce.
  const origin = req.headers.get("Origin");
  if (origin && !isAllowedOrigin(origin, env)) {
    return textResponse(403, "origin not allowed", req, env);
  }

  if (req.method === "OPTIONS") {
    return new Response(null, { status: 204, headers: corsHeaders(req, env) });
  }
  if (req.method !== "POST") {
    return textResponse(405, "method not allowed", req, env);
  }

  // Rate limit runs before the cache lookup: cached responses still consume
  // budget. This is intentional — the cache is a CPU optimization, not a
  // bypass for abuse controls.
  if (env.RATE_LIMITER) {
    const key = req.headers.get("cf-connecting-ip") || "unknown";
    const { success } = await env.RATE_LIMITER.limit({ key });
    if (!success) return textResponse(429, "rate limit exceeded", req, env);
  }

  const ct = req.headers.get("Content-Type") || "";
  if (!isAllowedContentType(ct)) {
    return textResponse(415, "unsupported media type", req, env);
  }

  let filename = "input";
  let bodyBytes;

  try {
    if (ct.toLowerCase().startsWith("multipart/form-data")) {
      const form = await req.formData();
      let file = null;
      for (const value of form.values()) {
        if (value instanceof File) {
          file = value;
          break;
        }
      }
      if (!file) return textResponse(400, "missing file part", req, env);
      if (file.size > MAX_BODY_BYTES) return textResponse(413, "file too large", req, env);
      filename = file.name || "input";
      bodyBytes = await file.arrayBuffer();
    } else {
      const url = new URL(req.url);
      filename = url.searchParams.get("filename") || "input";
      bodyBytes = await readWithCap(req);
    }
  } catch (e) {
    if (e && e.statusCode === 413) return textResponse(413, "request body too large", req, env);
    return textResponse(400, "could not read body", req, env);
  }

  // Cache lookup. The key mixes (workerVersion, filename, bodyBytes) so a new
  // deploy invalidates prior entries and multipart/raw paths share one cache.
  const cache = caches.default;
  const cacheKey = await lintCacheKey(req, env, filename, bodyBytes);
  const hit = await cache.match(cacheKey);
  if (hit) {
    return jsonResponse(200, await hit.text(), req, env, { [LINT_CACHE_HEADER]: "hit" });
  }

  await ensureWasm();
  const src = new TextDecoder().decode(bodyBytes);
  const out = globalThis.voidsliceLint(filename, src);
  const body = typeof out === "string" ? out : JSON.stringify(out);
  const cacheable = new Response(body, {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": `public, max-age=${LINT_CACHE_TTL_SECONDS}`,
    },
  });
  await cache.put(cacheKey, cacheable);
  return jsonResponse(200, body, req, env, { [LINT_CACHE_HEADER]: "miss" });
}

export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const start = Date.now();
    let resp;
    try {
      switch (url.pathname) {
        case "/health":
          if (req.method === "GET") {
            resp = healthResponse();
          } else {
            resp = textResponse(405, "method not allowed", req, env);
          }
          break;
        case "/lint":
          resp = await handleLint(req, env);
          break;
        default:
          resp = textResponse(404, "not found", req, env);
      }
    } catch (e) {
      console.error("worker error", e?.stack || e);
      resp = textResponse(500, "internal error", req, env);
    }
    console.log(
      JSON.stringify({
        method: req.method,
        path: url.pathname,
        status: resp.status,
        duration_ms: Date.now() - start,
      }),
    );
    return resp;
  },
};
