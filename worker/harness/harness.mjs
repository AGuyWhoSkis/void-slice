// WASM-boundary harness (M8.1).
//
// Loads worker/voidslice.wasm directly via worker/wasm_exec.js — no Worker, no
// HTTP, no frontend — and asserts that the Go→JS export contract matches what
// the native CLI emits via the same report.RenderJSON path. Any divergence is
// localized to a fixture and a JSON field, so failures point at the wasm
// boundary itself rather than a layer above it.
//
// Run via `make wasm-harness` from the repo root.

import { readFile, writeFile, rm } from "node:fs/promises";
import { mkdtempSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { tmpdir } from "node:os";

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, "..", "..");
const wasmPath = resolve(repoRoot, "worker", "voidslice.wasm");
const shimPath = resolve(repoRoot, "worker", "wasm_exec.js");

// Each fixture's `path` is the string passed to BOTH wasm (as the `filename`
// argument) and the CLI (as argv). Same string → same `file` field in JSON,
// so comparisons stay meaningful.
const FIXTURES = [
  {
    name: "clean",
    path: "testdata/corpus-mini/d2/game1/generated.decls.gamelogicmanager.ui.gamelogic.manager..gamelogicmanager.decl",
    reference: "cli",
  },
  {
    name: "validate-warning",
    path: "testdata/broken/count-mismatch.decl",
    reference: "cli",
  },
  {
    name: "scan-error",
    path: "testdata/broken/missing-semicolon.decl",
    reference: "cli",
  },
  {
    name: "empty-input",
    path: "empty.decl",
    inlineSrc: "",
    reference: { file: "empty.decl", diagnostics: [] },
  },
];

const WRONG_ARG_EXPECTED = {
  file: "",
  diagnostics: [
    {
      line: 0,
      col: 0,
      severity: "error",
      code: "WORKER_HARNESS",
      message: "voidsliceLint requires (filename, src)",
    },
  ],
};

async function loadWasm() {
  // wasm_exec.js is a side-effecting script that mutates globalThis. Eval it
  // in the global scope so it registers globalThis.Go.
  const shim = await readFile(shimPath, "utf8");
  // eslint-disable-next-line no-new-func
  new Function(shim).call(globalThis);
  if (typeof globalThis.Go !== "function") {
    throw new Error("wasm_exec.js did not register globalThis.Go");
  }

  const bytes = await readFile(wasmPath);
  const go = new globalThis.Go();
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  // go.run(instance) never resolves — main() calls select{}. Let it run; one
  // microtask tick is enough for the export to register.
  go.run(instance);
  await new Promise((r) => setTimeout(r, 0));
  if (typeof globalThis.voidsliceLint !== "function") {
    throw new Error("voidsliceLint export missing after WASM init");
  }
}

function runCLI(relPath, cwd) {
  const r = spawnSync("go", ["run", "./cmd/voidslice", "lint", relPath, "--json"], {
    cwd,
    encoding: "utf8",
  });
  // The CLI exits 1 when it emits any diagnostic at severity=error; that's a
  // normal outcome for the broken/* fixtures, not a harness error. We only
  // care that stdout is valid JSON.
  if (r.status !== 0 && r.status !== 1) {
    throw new Error(
      `go run ./cmd/voidslice lint exited ${r.status} for ${relPath}: ${r.stderr}`,
    );
  }
  return r.stdout;
}

function compare(name, wasmJSON, refJSON) {
  let w;
  let r;
  try {
    w = JSON.parse(wasmJSON);
  } catch (err) {
    return `${name}: wasm output is not valid JSON: ${err.message}`;
  }
  try {
    r = typeof refJSON === "string" ? JSON.parse(refJSON) : refJSON;
  } catch (err) {
    return `${name}: reference output is not valid JSON: ${err.message}`;
  }

  if (w.file !== r.file) {
    return `${name}: file: wasm=${JSON.stringify(w.file)} reference=${JSON.stringify(r.file)}`;
  }
  const wd = w.diagnostics || [];
  const rd = r.diagnostics || [];
  if (wd.length !== rd.length) {
    return `${name}: diagnostics.length: wasm=${wd.length} reference=${rd.length}`;
  }
  const fields = ["line", "col", "severity", "code", "message"];
  for (let i = 0; i < wd.length; i++) {
    for (const f of fields) {
      if (wd[i]?.[f] !== rd[i]?.[f]) {
        return `${name}: diagnostics[${i}].${f}: wasm=${JSON.stringify(wd[i]?.[f])} reference=${JSON.stringify(rd[i]?.[f])}`;
      }
    }
  }
  return null;
}

async function main() {
  await loadWasm();

  const failures = [];
  let scratch = null;

  for (const fx of FIXTURES) {
    let src;
    let cliCwd;
    if (fx.inlineSrc !== undefined) {
      src = fx.inlineSrc;
      // Need a real file on disk so the CLI can read it; place it in a
      // scratch dir whose layout makes `fx.path` resolve correctly.
      if (!scratch) {
        scratch = mkdtempSync(resolve(tmpdir(), "voidslice-harness-"));
      }
      await writeFile(resolve(scratch, fx.path), src);
      cliCwd = scratch;
    } else {
      src = await readFile(resolve(repoRoot, fx.path), "utf8");
      cliCwd = repoRoot;
    }

    const wasmOut = globalThis.voidsliceLint(fx.path, src);

    let reference;
    if (fx.reference === "cli") {
      reference = runCLI(fx.path, cliCwd);
    } else {
      reference = fx.reference;
    }

    const fail = compare(fx.name, wasmOut, reference);
    if (fail) {
      failures.push(fail);
      console.log(`FAIL ${fx.name}`);
    } else {
      console.log(`OK   ${fx.name}`);
    }
  }

  // Wrong-arg-count case: invoke the export with zero JS arguments and
  // assert it returns the WORKER_HARNESS error template verbatim.
  const wrongArgOut = globalThis.voidsliceLint();
  const wrongArgFail = compare("wrong-arg-count", wrongArgOut, WRONG_ARG_EXPECTED);
  if (wrongArgFail) {
    failures.push(wrongArgFail);
    console.log("FAIL wrong-arg-count");
  } else {
    console.log("OK   wrong-arg-count");
  }

  if (scratch) await rm(scratch, { recursive: true, force: true });

  if (failures.length > 0) {
    console.error("");
    for (const f of failures) console.error(`FIXTURE ${f}`);
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(`harness: ${err.stack || err.message}`);
  process.exit(2);
});
