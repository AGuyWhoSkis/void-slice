// Worker-glue harness (M8.2).
//
// Boots worker/index.js inside Miniflare 3 (workerd) in-process — no
// `wrangler dev`, no preview URL, no Cloudflare account — and asserts every
// branch of handleLint and the top-level router. For 200-cases the response
// body is compared against the WASM-boundary oracle (the same export M8.1
// pins). Failures are localized to a case name and a single field, so a
// divergence here points at the Worker glue, not the linter or the wasm
// boundary.
//
// Run via `make worker-harness` from the repo root.

import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { Miniflare } from "miniflare";

import { FIXTURES } from "./fixtures.mjs";
import { loadWasm } from "./wasm-loader.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, "..", "..");
const workerScriptPath = resolve(repoRoot, "worker", "index.js");

// Mirrors worker/index.js. Test 8 sends MAX_BODY_BYTES + 1; if the constant
// in the Worker changes, this harness must change too — that coupling is the
// point.
const MAX_BODY_BYTES = 1 << 20;
const COMPATIBILITY_DATE = "2025-09-01";
const ORIGIN_BASE = "http://harness.local";
const SPECIFIC_ORIGIN = "https://example.com";
const SUFFIX = ".example.com";
const SUFFIX_ALLOWED_ORIGIN = "https://pr-7.example.com";
// "evilexample.com" without a leading dot is the label-boundary attack:
// it endsWith "example.com" but must NOT match suffix ".example.com".
const SUFFIX_BOUNDARY_ATTACK = "https://evilexample.com";
const FOREIGN_ORIGIN = "https://evil.example";

// `empty-input` is the cheapest fixture — no disk read, no CLI shell-out.
// It's the right pick for cases that are about Worker glue, not linter
// content.
const VALID = FIXTURES.find((fx) => fx.name === "empty-input");
if (!VALID || VALID.inlineSrc === undefined) {
  throw new Error("fixtures.mjs: missing inline empty-input fixture");
}
const VALID_FILENAME = VALID.path;
const VALID_SRC = VALID.inlineSrc;

// --- WASM oracle ---------------------------------------------------------
// Loads the WASM export into this harness's own globalThis so we can compute
// expected bodies for 200-cases without a second subprocess.

function oracle(filename, src) {
  return globalThis.voidsliceLint(filename, src);
}

// --- Miniflare instance helpers ------------------------------------------

function newMiniflare({
  allowedOrigins = "",
  allowedSuffixes = "",
  rateLimit,
  workerVersion = "harness-v1",
} = {}) {
  const opts = {
    scriptPath: workerScriptPath,
    modules: true,
    modulesRules: [
      { type: "ESModule", include: ["**/*.js"], fallthrough: true },
      { type: "CompiledWasm", include: ["**/*.wasm"], fallthrough: true },
    ],
    compatibilityDate: COMPATIBILITY_DATE,
    bindings: {
      ALLOWED_ORIGINS: allowedOrigins,
      ALLOWED_ORIGIN_SUFFIXES: allowedSuffixes,
      WORKER_VERSION: workerVersion,
    },
  };
  if (rateLimit) {
    opts.ratelimits = {
      RATE_LIMITER: {
        simple: { limit: rateLimit.limit, period: rateLimit.period ?? 60 },
      },
    };
  }
  return new Miniflare(opts);
}

// --- Assertion helpers ---------------------------------------------------

const failures = [];

function record(caseName, field, got, want) {
  failures.push(
    `case=${caseName} field=${field} got=${JSON.stringify(got)} want=${JSON.stringify(want)}`,
  );
}

function expectStatus(caseName, resp, want) {
  if (resp.status !== want) record(caseName, "status", resp.status, want);
}

function expectHeader(caseName, resp, name, want) {
  const got = resp.headers.get(name);
  if (got !== want) record(caseName, `header.${name}`, got, want);
}

function expectHeaderAbsent(caseName, resp, name) {
  const got = resp.headers.get(name);
  if (got !== null) record(caseName, `header.${name}`, got, null);
}

function expectJsonBodyEquals(caseName, gotBody, wantBody) {
  // Canonicalize JSON to ignore whitespace.
  let g, w;
  try {
    g = JSON.parse(gotBody);
  } catch (err) {
    record(caseName, "body.json", gotBody.slice(0, 120), `valid JSON (${err.message})`);
    return;
  }
  try {
    w = JSON.parse(wantBody);
  } catch (err) {
    record(caseName, "body.json.want", wantBody.slice(0, 120), `valid JSON (${err.message})`);
    return;
  }
  const gs = JSON.stringify(g);
  const ws = JSON.stringify(w);
  if (gs !== ws) record(caseName, "body", gs, ws);
}

// --- Cases ---------------------------------------------------------------

async function runNoAllowlistCases(mf) {
  // No allowlist configured. All requests in this family omit the Origin
  // header (curl/server-to-server style), so the allowlist gate doesn't
  // fire — they should all proceed normally without any ACAO. Vary: Origin
  // is still emitted so caches differentiate.

  // 1. health-get — /health is intentionally public; ACAO: * regardless.
  {
    const name = "health-get";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/health`, { method: "GET" });
    expectStatus(name, r, 200);
    expectHeader(name, r, "Access-Control-Allow-Origin", "*");
    const body = await r.text();
    expectJsonBodyEquals(name, body, '{"status":"ok"}');
  }

  // 2. health-non-get
  {
    const name = "health-non-get";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/health`, { method: "PUT" });
    expectStatus(name, r, 405);
    await r.text();
  }

  // 3. lint-octet-stream — no Origin → no allowlist gate → 200.
  {
    const name = "lint-octet-stream";
    const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/octet-stream" },
      body: VALID_SRC,
    });
    expectStatus(name, r, 200);
    expectHeaderAbsent(name, r, "Access-Control-Allow-Origin");
    expectHeader(name, r, "Vary", "Origin");
    const body = await r.text();
    expectJsonBodyEquals(name, body, oracle(VALID_FILENAME, VALID_SRC));
  }

  // 5. lint-options-no-origin — preflight without Origin. Browsers don't
  // send this shape; covers a misbehaving client. Worker returns 204 with
  // no permissive ACAO, only Vary.
  {
    const name = "lint-options-no-origin";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, { method: "OPTIONS" });
    expectStatus(name, r, 204);
    expectHeaderAbsent(name, r, "Access-Control-Allow-Origin");
    expectHeader(name, r, "Vary", "Origin");
  }

  // 6. lint-get-rejected
  {
    const name = "lint-get-rejected";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, { method: "GET" });
    expectStatus(name, r, 405);
    await r.text();
  }

  // 7. lint-bad-content-type
  {
    const name = "lint-bad-content-type";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, {
      method: "POST",
      headers: { "Content-Type": "image/png" },
      body: "x",
    });
    expectStatus(name, r, 415);
    await r.text();
  }

  // 8. lint-body-too-large
  {
    const name = "lint-body-too-large";
    const big = new Uint8Array(MAX_BODY_BYTES + 1).fill(0x61);
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, {
      method: "POST",
      headers: { "Content-Type": "application/octet-stream" },
      body: big,
    });
    expectStatus(name, r, 413);
    await r.text();
  }

  // 9. lint-multipart-with-file
  {
    const name = "lint-multipart-with-file";
    const fd = new FormData();
    fd.append("file", new Blob([VALID_SRC], { type: "text/plain" }), VALID_FILENAME);
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, { method: "POST", body: fd });
    expectStatus(name, r, 200);
    const body = await r.text();
    expectJsonBodyEquals(name, body, oracle(VALID_FILENAME, VALID_SRC));
  }

  // 10. lint-multipart-no-file
  {
    const name = "lint-multipart-no-file";
    const fd = new FormData();
    fd.append("note", "no file part here");
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, { method: "POST", body: fd });
    expectStatus(name, r, 400);
    await r.text();
  }

  // 11. unknown-path
  {
    const name = "unknown-path";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/nope`, { method: "GET" });
    expectStatus(name, r, 404);
    await r.text();
  }

  // 14. cors-no-origin-on-200 — POST without Origin succeeds and sends no
  // permissive ACAO. Vary: Origin is set so a downstream cache that later
  // sees an Origin'd request doesn't serve this response back.
  {
    const name = "cors-no-origin-on-200";
    const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/octet-stream" },
      body: VALID_SRC,
    });
    expectStatus(name, r, 200);
    expectHeaderAbsent(name, r, "Access-Control-Allow-Origin");
    expectHeader(name, r, "Vary", "Origin");
    await r.text();
  }
}

async function runSpecificOriginCases(mf) {
  // 4. lint-options-allowed
  {
    const name = "lint-options-allowed";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, {
      method: "OPTIONS",
      headers: { Origin: SPECIFIC_ORIGIN },
    });
    expectStatus(name, r, 204);
    expectHeader(name, r, "Access-Control-Allow-Origin", SPECIFIC_ORIGIN);
    expectHeader(name, r, "Access-Control-Allow-Methods", "POST, GET, OPTIONS");
    expectHeader(name, r, "Access-Control-Allow-Headers", "Content-Type");
    expectHeader(name, r, "Vary", "Origin");
  }

  // 15. cors-specific-origin
  {
    const name = "cors-specific-origin";
    const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/octet-stream",
        Origin: SPECIFIC_ORIGIN,
      },
      body: VALID_SRC,
    });
    expectStatus(name, r, 200);
    expectHeader(name, r, "Access-Control-Allow-Origin", SPECIFIC_ORIGIN);
    expectHeader(name, r, "Vary", "Origin");
    await r.text();
  }

  // 16. lint-options-foreign — disallowed Origin on preflight is rejected
  // before the 204 path, with no permissive headers. Browser drops it.
  {
    const name = "lint-options-foreign";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, {
      method: "OPTIONS",
      headers: { Origin: FOREIGN_ORIGIN },
    });
    expectStatus(name, r, 403);
    expectHeaderAbsent(name, r, "Access-Control-Allow-Origin");
    await r.text();
  }

  // 17. cors-foreign-origin — POST with disallowed Origin short-circuits to
  // 403 *before* any linter or rate-limiter work. This is the budget defense
  // against "simple" POSTs (Content-Type: text/plain) that skip preflight.
  {
    const name = "cors-foreign-origin";
    const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/octet-stream",
        Origin: FOREIGN_ORIGIN,
      },
      body: VALID_SRC,
    });
    expectStatus(name, r, 403);
    expectHeaderAbsent(name, r, "Access-Control-Allow-Origin");
    await r.text();
  }
}

async function runSuffixCases(mf) {
  // 18. suffix-options-allowed — Origin matches a leading-dot suffix.
  {
    const name = "suffix-options-allowed";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, {
      method: "OPTIONS",
      headers: { Origin: SUFFIX_ALLOWED_ORIGIN },
    });
    expectStatus(name, r, 204);
    expectHeader(name, r, "Access-Control-Allow-Origin", SUFFIX_ALLOWED_ORIGIN);
    expectHeader(name, r, "Vary", "Origin");
  }

  // 19. suffix-post-allowed
  {
    const name = "suffix-post-allowed";
    const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/octet-stream",
        Origin: SUFFIX_ALLOWED_ORIGIN,
      },
      body: VALID_SRC,
    });
    expectStatus(name, r, 200);
    expectHeader(name, r, "Access-Control-Allow-Origin", SUFFIX_ALLOWED_ORIGIN);
    expectHeader(name, r, "Vary", "Origin");
    await r.text();
  }

  // 20. suffix-options-boundary — leading-dot enforces a label boundary so
  // "evilexample.com" must NOT match suffix ".example.com".
  {
    const name = "suffix-options-boundary";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, {
      method: "OPTIONS",
      headers: { Origin: SUFFIX_BOUNDARY_ATTACK },
    });
    expectStatus(name, r, 403);
    expectHeaderAbsent(name, r, "Access-Control-Allow-Origin");
    await r.text();
  }

  // 21. suffix-post-boundary
  {
    const name = "suffix-post-boundary";
    const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/octet-stream",
        Origin: SUFFIX_BOUNDARY_ATTACK,
      },
      body: VALID_SRC,
    });
    expectStatus(name, r, 403);
    expectHeaderAbsent(name, r, "Access-Control-Allow-Origin");
    await r.text();
  }

  // 22. suffix-http-rejected — suffix matching requires https; an http
  // Origin pointing at a covered host is denied.
  {
    const name = "suffix-http-rejected";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, {
      method: "OPTIONS",
      headers: { Origin: "http://pr-7.example.com" },
    });
    expectStatus(name, r, 403);
    expectHeaderAbsent(name, r, "Access-Control-Allow-Origin");
    await r.text();
  }
}

async function runRateLimitBlockedCases(mf) {
  // 12. rate-limit-429 — RATE_LIMITER bound with limit:1, drained on the
  // first call so the second is blocked. Miniflare's zod schema rejects
  // limit:0, so we drain explicitly. Both calls go through handleLint with
  // a stable cf-connecting-ip key (Miniflare's default).
  const name = "rate-limit-429";
  const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
  const init = {
    method: "POST",
    headers: { "Content-Type": "application/octet-stream" },
    body: VALID_SRC,
  };
  const drain = await mf.dispatchFetch(url, init);
  await drain.text();
  if (drain.status !== 200) {
    record(name, "drain.status", drain.status, 200);
    return;
  }
  const r = await mf.dispatchFetch(url, init);
  expectStatus(name, r, 429);
  await r.text();
}

async function runCacheCases(mf) {
  // Repeat-lint cache: same (filename, body) on the same Worker version
  // hashes to the same cache key, so the second POST short-circuits before
  // ensureWasm() and returns X-Voidslice-Cache: hit. Different filename or
  // body on the same version → miss. /health never carries the header.
  const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
  const init = {
    method: "POST",
    headers: { "Content-Type": "application/octet-stream" },
    body: VALID_SRC,
  };

  let firstBody;
  {
    const name = "cache-first-miss";
    const r = await mf.dispatchFetch(url, init);
    expectStatus(name, r, 200);
    expectHeader(name, r, "X-Voidslice-Cache", "miss");
    firstBody = await r.text();
    expectJsonBodyEquals(name, firstBody, oracle(VALID_FILENAME, VALID_SRC));
  }

  {
    const name = "cache-second-hit";
    const r = await mf.dispatchFetch(url, init);
    expectStatus(name, r, 200);
    expectHeader(name, r, "X-Voidslice-Cache", "hit");
    const body = await r.text();
    if (body !== firstBody) record(name, "body", body.slice(0, 120), firstBody.slice(0, 120));
  }

  {
    const name = "cache-different-body";
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/octet-stream" },
      body: VALID_SRC + "\n",
    });
    expectStatus(name, r, 200);
    expectHeader(name, r, "X-Voidslice-Cache", "miss");
    await r.text();
  }

  {
    const name = "cache-different-filename";
    const altUrl = `${ORIGIN_BASE}/lint?filename=other.entities`;
    const r = await mf.dispatchFetch(altUrl, init);
    expectStatus(name, r, 200);
    expectHeader(name, r, "X-Voidslice-Cache", "miss");
    await r.text();
  }

  {
    const name = "cache-health-uncached";
    const a = await mf.dispatchFetch(`${ORIGIN_BASE}/health`, { method: "GET" });
    expectHeaderAbsent(name, a, "X-Voidslice-Cache");
    await a.text();
    const b = await mf.dispatchFetch(`${ORIGIN_BASE}/health`, { method: "GET" });
    expectHeaderAbsent(name, b, "X-Voidslice-Cache");
    await b.text();
  }
}

async function runCacheRateLimitCases(mf) {
  // Cache hits do not bypass the rate limiter — rate limit fires before the
  // cache lookup. First POST primes the cache and drains the limit; the
  // second POST is what would have been a hit but is blocked at 429 before
  // any cache work runs.
  const name = "cache-vs-rate-limit";
  const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
  const init = {
    method: "POST",
    headers: { "Content-Type": "application/octet-stream" },
    body: VALID_SRC,
  };
  const drain = await mf.dispatchFetch(url, init);
  await drain.text();
  if (drain.status !== 200) {
    record(name, "drain.status", drain.status, 200);
    return;
  }
  const r = await mf.dispatchFetch(url, init);
  expectStatus(name, r, 429);
  expectHeaderAbsent(name, r, "X-Voidslice-Cache");
  await r.text();
}

async function runRateLimitUnboundCases(mf) {
  // 13. rate-limit-200 — RATE_LIMITER unbound: same request as case 12 must
  // succeed. This pins the toggle, not the policy.
  const name = "rate-limit-200";
  const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
  const r = await mf.dispatchFetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/octet-stream" },
    body: VALID_SRC,
  });
  expectStatus(name, r, 200);
  await r.text();
}

// --- main ----------------------------------------------------------------

async function main() {
  await loadWasm();

  const families = [
    { name: "no-allowlist", build: () => newMiniflare(), run: runNoAllowlistCases },
    {
      name: "specific-origin",
      build: () => newMiniflare({ allowedOrigins: SPECIFIC_ORIGIN }),
      run: runSpecificOriginCases,
    },
    {
      name: "suffix",
      build: () => newMiniflare({ allowedSuffixes: SUFFIX }),
      run: runSuffixCases,
    },
    {
      name: "rate-limit-blocked",
      // limit: 0 → every call to env.RATE_LIMITER.limit returns success:false.
      build: () => newMiniflare({ rateLimit: { limit: 1 } }),
      run: runRateLimitBlockedCases,
    },
    {
      name: "rate-limit-unbound",
      build: () => newMiniflare(),
      run: runRateLimitUnboundCases,
    },
    {
      name: "cache",
      build: () => newMiniflare(),
      run: runCacheCases,
    },
    {
      name: "cache-rate-limit",
      build: () => newMiniflare({ rateLimit: { limit: 1 } }),
      run: runCacheRateLimitCases,
    },
  ];

  for (const fam of families) {
    const before = failures.length;
    const mf = fam.build();
    try {
      await fam.run(mf);
    } finally {
      await mf.dispose();
    }
    console.log(`${failures.length === before ? "OK  " : "FAIL"} family=${fam.name}`);
  }

  if (failures.length > 0) {
    console.error("");
    for (const f of failures) console.error(`FAIL ${f}`);
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(`worker-harness: ${err.stack || err.message}`);
  process.exit(2);
});
