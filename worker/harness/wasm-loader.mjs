// Shared WASM loader for the harnesses in this directory.
//
// Reads worker/wasm_exec.js and worker/voidslice.wasm, evals the shim into
// the caller's globalThis to register `Go`, instantiates the wasm, and waits
// one microtask so the Go side can register `globalThis.voidsliceLint`.
//
// All three callers (harness.mjs, worker-harness.mjs, oracle.mjs) need
// exactly this sequence; keeping it in one place means a wasm_exec.js or
// export-name change touches one file, not three.

import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, "..", "..");
const wasmPath = resolve(repoRoot, "worker", "voidslice.wasm");
const shimPath = resolve(repoRoot, "worker", "wasm_exec.js");

export async function loadWasm() {
  // wasm_exec.js is a side-effecting script that mutates globalThis. Eval it
  // in global scope so it registers `globalThis.Go`.
  const shim = await readFile(shimPath, "utf8");
  // eslint-disable-next-line no-new-func
  new Function(shim).call(globalThis);
  if (typeof globalThis.Go !== "function") {
    throw new Error("wasm_exec.js did not register globalThis.Go");
  }

  const bytes = await readFile(wasmPath);
  const go = new globalThis.Go();
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  // go.run(instance) never resolves — main() calls select{}. One microtask
  // tick is enough for the export to register.
  go.run(instance);
  await new Promise((r) => setTimeout(r, 0));
  if (typeof globalThis.voidsliceLint !== "function") {
    throw new Error("voidsliceLint export missing after WASM init");
  }
}
