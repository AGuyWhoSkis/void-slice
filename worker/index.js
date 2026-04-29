// Cloudflare Worker entry point. Wraps the Go-compiled WASM module exposed
// at globalThis.voidsliceLint by cmd/voidslice-wasm/main.go and serves the
// same /health + /lint API as internal/server.
//
// Wrangler's [wasm_modules] binding (see wrangler.toml) provides the
// compiled .wasm as a WebAssembly.Module on `env.VOIDSLICE_WASM`.

import "./wasm_exec.js";

const MAX_BODY_BYTES = 1 << 20; // 1 MiB — see kanban/done/v2/T26-linter-resource-profile.md
const ALLOW_METHODS = "POST, GET, OPTIONS";
const ALLOW_HEADERS = "Content-Type";

let wasmReady = null;

function ensureWasm(env) {
  if (wasmReady) return wasmReady;
  wasmReady = (async () => {
    // eslint-disable-next-line no-undef
    const go = new Go();
    const instance = await WebAssembly.instantiate(env.VOIDSLICE_WASM, go.importObject);
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

function corsHeaders(origin) {
  const allowed = origin || "*";
  const headers = {
    "Access-Control-Allow-Origin": allowed,
    "Access-Control-Allow-Methods": ALLOW_METHODS,
    "Access-Control-Allow-Headers": ALLOW_HEADERS,
  };
  if (allowed !== "*") headers["Vary"] = "Origin";
  return headers;
}

function jsonResponse(status, body, env) {
  return new Response(typeof body === "string" ? body : JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
      ...corsHeaders(env.ALLOWED_ORIGIN),
    },
  });
}

function textResponse(status, body, env) {
  return new Response(body, {
    status,
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      ...corsHeaders(env.ALLOWED_ORIGIN),
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

async function handleLint(req, env) {
  if (req.method === "OPTIONS") {
    return new Response(null, { status: 204, headers: corsHeaders(env.ALLOWED_ORIGIN) });
  }
  if (req.method !== "POST") {
    return textResponse(405, "method not allowed", env);
  }

  const ct = req.headers.get("Content-Type") || "";
  if (!isAllowedContentType(ct)) {
    return textResponse(415, "unsupported media type", env);
  }

  let filename = "input";
  let src = "";

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
      if (!file) return textResponse(400, "missing file part", env);
      if (file.size > MAX_BODY_BYTES) return textResponse(413, "file too large", env);
      filename = file.name || "input";
      src = await file.text();
    } else {
      const url = new URL(req.url);
      filename = url.searchParams.get("filename") || "input";
      const buf = await readWithCap(req);
      src = new TextDecoder().decode(buf);
    }
  } catch (e) {
    if (e && e.statusCode === 413) return textResponse(413, "request body too large", env);
    return textResponse(400, "could not read body", env);
  }

  await ensureWasm(env);
  const out = globalThis.voidsliceLint(filename, src);
  return jsonResponse(200, out, env);
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
            resp = jsonResponse(200, { status: "ok" }, env);
          } else {
            resp = textResponse(405, "method not allowed", env);
          }
          break;
        case "/lint":
          resp = await handleLint(req, env);
          break;
        default:
          resp = textResponse(404, "not found", env);
      }
    } catch (e) {
      console.error("worker error", e?.stack || e);
      resp = textResponse(500, "internal error", env);
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
