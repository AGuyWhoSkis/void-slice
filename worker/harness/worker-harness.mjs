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

function newMiniflare({ allowedOrigin = "*", rateLimit } = {}) {
  const opts = {
    scriptPath: workerScriptPath,
    modules: true,
    modulesRules: [
      { type: "ESModule", include: ["**/*.js"], fallthrough: true },
      { type: "CompiledWasm", include: ["**/*.wasm"], fallthrough: true },
    ],
    compatibilityDate: COMPATIBILITY_DATE,
    bindings: { ALLOWED_ORIGIN: allowedOrigin },
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

async function runDefaultCases(mf) {
  // 1. health-get
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

  // 3. lint-octet-stream
  {
    const name = "lint-octet-stream";
    const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/octet-stream" },
      body: VALID_SRC,
    });
    expectStatus(name, r, 200);
    const body = await r.text();
    expectJsonBodyEquals(name, body, oracle(VALID_FILENAME, VALID_SRC));
  }

  // 5. lint-options-wildcard
  {
    const name = "lint-options-wildcard";
    const r = await mf.dispatchFetch(`${ORIGIN_BASE}/lint`, { method: "OPTIONS" });
    expectStatus(name, r, 204);
    expectHeader(name, r, "Access-Control-Allow-Origin", "*");
    expectHeader(name, r, "Access-Control-Allow-Methods", "POST, GET, OPTIONS");
    expectHeader(name, r, "Access-Control-Allow-Headers", "Content-Type");
    expectHeaderAbsent(name, r, "Vary");
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

  // 14. cors-wildcard-on-200
  {
    const name = "cors-wildcard-on-200";
    const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(VALID_FILENAME)}`;
    const r = await mf.dispatchFetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/octet-stream" },
      body: VALID_SRC,
    });
    expectStatus(name, r, 200);
    expectHeader(name, r, "Access-Control-Allow-Origin", "*");
    expectHeaderAbsent(name, r, "Vary");
    await r.text();
  }
}

async function runSpecificOriginCases(mf) {
  // 4. lint-options
  {
    const name = "lint-options";
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
      headers: { "Content-Type": "application/octet-stream" },
      body: VALID_SRC,
    });
    expectStatus(name, r, 200);
    expectHeader(name, r, "Access-Control-Allow-Origin", SPECIFIC_ORIGIN);
    expectHeader(name, r, "Vary", "Origin");
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
    { name: "default", build: () => newMiniflare(), run: runDefaultCases },
    {
      name: "specific-origin",
      build: () => newMiniflare({ allowedOrigin: SPECIFIC_ORIGIN }),
      run: runSpecificOriginCases,
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
