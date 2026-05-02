// Differential oracle (M8.4).
//
// Runs each fixture through four layers and reports the first boundary where
// outputs diverge. The chain is:
//
//   1. native      — bin/voidslice lint <file> --json (internal/lint via CLI)
//   2. wasm        — globalThis.voidsliceLint(filename, src), loaded directly
//   3. worker      — POST /lint to worker/index.js running in Miniflare
//   4. frontend    — replica of web/src/api.ts's lintFile() against the same
//                    Miniflare instance
//
// Boundaries between adjacent layers (1↔2, 2↔3, 3↔4) are what divergence
// names. The oracle does not redefine layer-internal assertions — those stay
// in worker/harness/{harness,worker-harness}.mjs and web/src/api.test.ts.
// This script only diffs the four final outputs.
//
// Run via `make oracle` from the repo root.

import { mkdtempSync } from "node:fs";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { tmpdir } from "node:os";
import { Miniflare } from "miniflare";

import { FIXTURES } from "./fixtures.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, "..", "..");
const wasmPath = resolve(repoRoot, "worker", "voidslice.wasm");
const shimPath = resolve(repoRoot, "worker", "wasm_exec.js");
const workerScriptPath = resolve(repoRoot, "worker", "index.js");

const COMPATIBILITY_DATE = "2025-09-01";
const ORIGIN_BASE = "http://oracle.local";

// --- Layer 1: native (CLI built once) ------------------------------------

function buildCLI(outDir) {
  const binPath = resolve(outDir, "voidslice");
  const r = spawnSync("go", ["build", "-o", binPath, "./cmd/voidslice"], {
    cwd: repoRoot,
    encoding: "utf8",
  });
  if (r.status !== 0) {
    throw new Error(`go build failed: ${r.stderr || r.stdout}`);
  }
  return binPath;
}

function runNative(binPath, relPath, cwd) {
  const r = spawnSync(binPath, ["lint", relPath, "--json"], { cwd, encoding: "utf8" });
  // Exit 1 is normal when fixtures emit error-severity diagnostics; only
  // crashes (other non-zero) should fail the harness.
  if (r.status !== 0 && r.status !== 1) {
    throw new Error(`native CLI exited ${r.status} for ${relPath}: ${r.stderr}`);
  }
  return r.stdout;
}

// --- Layer 2: wasm (loaded into this process's globalThis) ---------------

async function loadWasm() {
  const shim = await readFile(shimPath, "utf8");
  // wasm_exec.js mutates globalThis. Eval it in global scope.
  // eslint-disable-next-line no-new-func
  new Function(shim).call(globalThis);
  if (typeof globalThis.Go !== "function") {
    throw new Error("wasm_exec.js did not register globalThis.Go");
  }
  const bytes = await readFile(wasmPath);
  const go = new globalThis.Go();
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  // go.run() never resolves — main() calls select{}. One microtask is enough
  // for the export to register.
  go.run(instance);
  await new Promise((r) => setTimeout(r, 0));
  if (typeof globalThis.voidsliceLint !== "function") {
    throw new Error("voidsliceLint export missing after WASM init");
  }
}

function runWasm(filename, src) {
  return globalThis.voidsliceLint(filename, src);
}

// --- Layer 3: worker (Miniflare, in-process) -----------------------------

function newMiniflare() {
  return new Miniflare({
    scriptPath: workerScriptPath,
    modules: true,
    modulesRules: [
      { type: "ESModule", include: ["**/*.js"], fallthrough: true },
      { type: "CompiledWasm", include: ["**/*.wasm"], fallthrough: true },
    ],
    compatibilityDate: COMPATIBILITY_DATE,
    bindings: { ALLOWED_ORIGIN: "*" },
  });
}

async function runWorker(mf, filename, src) {
  const url = `${ORIGIN_BASE}/lint?filename=${encodeURIComponent(filename)}`;
  const r = await mf.dispatchFetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/octet-stream" },
    body: src,
  });
  const body = await r.text();
  if (r.status !== 200) {
    throw new Error(`worker returned status ${r.status}: ${body.slice(0, 120)}`);
  }
  return body;
}

// --- Layer 4: frontend transport (replica of web/src/api.ts lintFile) ---
//
// MIRROR of web/src/api.ts's lintFile(). Coupled by inspection: same URL
// shape, method, header, body, JSON parse path. M8.3's vitest pins api.ts
// directly — if api.ts drifts from this shape, that harness fires first.

async function runFrontend(mfFetch, baseURL, filename, src) {
  const url = `${baseURL}/lint?filename=${encodeURIComponent(filename)}`;
  const res = await mfFetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/octet-stream" },
    body: src,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`lint failed (${res.status}): ${text || res.statusText}`);
  }
  // api.ts does res.json() then casts to LintResponse; mimic by JSON-parsing
  // the text so the result is a plain object comparable to the other layers.
  return JSON.stringify(await res.json());
}

// --- Diff -----------------------------------------------------------------

const DIAG_FIELDS = ["line", "col", "severity", "code", "message"];

function parseJSON(label, raw) {
  if (raw && typeof raw === "object") return raw;
  try {
    return JSON.parse(raw);
  } catch (err) {
    throw new Error(`${label}: invalid JSON: ${err.message}: ${String(raw).slice(0, 120)}`);
  }
}

// Returns null if equal, otherwise a string describing the first
// field-level mismatch.
function fieldDiff(a, b) {
  if (a.file !== b.file) {
    return `file: ${JSON.stringify(a.file)} vs ${JSON.stringify(b.file)}`;
  }
  const ad = a.diagnostics || [];
  const bd = b.diagnostics || [];
  if (ad.length !== bd.length) {
    return `diagnostics.length: ${ad.length} vs ${bd.length}`;
  }
  for (let i = 0; i < ad.length; i++) {
    for (const f of DIAG_FIELDS) {
      if (ad[i]?.[f] !== bd[i]?.[f]) {
        return `diagnostics[${i}].${f}: ${JSON.stringify(ad[i]?.[f])} vs ${JSON.stringify(bd[i]?.[f])}`;
      }
    }
  }
  return null;
}

// Walk the chain in order. Names the first boundary where two adjacent
// layers disagree. Returns null on full agreement, otherwise a string.
function chainDiff(name, results) {
  const order = ["native", "wasm", "worker", "frontend"];
  for (let i = 1; i < order.length; i++) {
    const from = order[i - 1];
    const to = order[i];
    const diff = fieldDiff(results[from], results[to]);
    if (diff) {
      return `${name}: diverged at boundary ${i} (${from} ↔ ${to}) on ${diff}`;
    }
  }
  return null;
}

// --- main ----------------------------------------------------------------

async function main() {
  const scratch = mkdtempSync(resolve(tmpdir(), "voidslice-oracle-"));
  const fixtureDir = resolve(scratch, "fixtures");
  await mkdir(fixtureDir, { recursive: true });

  const binPath = buildCLI(scratch);
  await loadWasm();
  const mf = newMiniflare();
  // Touch the worker once so wasm init in the Worker is paid before the
  // timed loop and any failure surfaces with a clear error.
  const ping = await mf.dispatchFetch(`${ORIGIN_BASE}/health`, { method: "GET" });
  await ping.text();
  if (ping.status !== 200) {
    await mf.dispose();
    await rm(scratch, { recursive: true, force: true });
    throw new Error(`worker /health returned ${ping.status} during warmup`);
  }
  // Real fetch handle bound to this Miniflare instance — passed to the
  // frontend replica so its transport actually crosses the worker layer.
  const mfFetch = mf.dispatchFetch.bind(mf);

  const failures = [];

  try {
    for (const fx of FIXTURES) {
      let src;
      let nativeCwd;
      if (fx.inlineSrc !== undefined) {
        src = fx.inlineSrc;
        await writeFile(resolve(fixtureDir, fx.path), src);
        nativeCwd = fixtureDir;
      } else {
        src = await readFile(resolve(repoRoot, fx.path), "utf8");
        nativeCwd = repoRoot;
      }

      const nativeRaw = runNative(binPath, fx.path, nativeCwd);
      const wasmRaw = runWasm(fx.path, src);
      const workerRaw = await runWorker(mf, fx.path, src);
      const frontendRaw = await runFrontend(mfFetch, ORIGIN_BASE, fx.path, src);

      const results = {
        native: parseJSON(`${fx.name}/native`, nativeRaw),
        wasm: parseJSON(`${fx.name}/wasm`, wasmRaw),
        worker: parseJSON(`${fx.name}/worker`, workerRaw),
        frontend: parseJSON(`${fx.name}/frontend`, frontendRaw),
      };

      const diff = chainDiff(fx.name, results);
      if (diff) {
        failures.push(diff);
        console.log(`FAIL ${fx.name}`);
      } else {
        console.log(`OK   ${fx.name}`);
      }
    }
  } finally {
    await mf.dispose();
    await rm(scratch, { recursive: true, force: true });
  }

  console.log("");
  if (failures.length === 0) {
    console.log(`oracle: all ${FIXTURES.length} inputs agree across native, wasm, worker, frontend`);
    return;
  }
  console.error(`oracle: ${failures.length}/${FIXTURES.length} inputs diverged`);
  for (const f of failures) console.error(`  ${f}`);
  process.exit(1);
}

main().catch((err) => {
  console.error(`oracle: ${err.stack || err.message}`);
  process.exit(2);
});
