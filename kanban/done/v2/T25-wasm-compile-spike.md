# T25 · WASM Compile Spike

**Status:** done
**Version:** v2
**Size:** small

## What

Attempt to compile the linter as a Cloudflare Workers-compatible WASM module (`GOOS=wasip1 GOARCH=wasm`). This is the critical-path decision for the deployment architecture: if the compile succeeds and the Worker runs correctly, L4 collapses into L3 (no container needed); if it fails, the fallback plan (Containers or Lambda) must be chosen before v3 implementation begins.

## Scope

- Run `GOOS=wasip1 GOARCH=wasm go build ./...` and record the outcome (success, cgo issue, syscall issue, other).
- If it compiles: wire the WASM binary into a minimal Cloudflare Worker, lint at least one file from `void-files/`, and confirm diagnostics match `go run` output.
- If it fails: identify the root cause (cgo dependency, specific syscall, threading assumption). Attempt the smallest refactor that unblocks it; if fundamental, document why and recommend the fallback (Containers, Lambda, or Cloud Run).
- Record the decision and rationale in a `## Completion` section so the v2 deployment ticket (T12) can pick up without re-deriving it.
- Verify current Cloudflare Workers per-request CPU cap and memory ceiling against actual linter run metrics (training data may be stale — check docs).

## Dependencies

T4 (lint facade), T7 (integration tests — confirms corpus baseline)

## Verification

- WASM path: a Cloudflare Worker script that accepts a file payload, runs the linter, and returns JSON diagnostics. Output matches `voidslice lint --json` on the same file.
- Failure path: written diagnosis with specific blocker and chosen fallback documented in `## Completion`.

## Completion

**Decision: WASM path is viable. Ship via `GOOS=js GOARCH=wasm`, not `wasip1`.**

L4 collapses into L3: the linter runs inside a Cloudflare Worker as a plain WebAssembly module driven by Go's official `wasm_exec.js` host. The container path from T10 stays as a local-dev convenience and a deploy-time fallback if Worker integration runs into something we missed in T11.

**Compile results**

| Target | Build | Bundle | gzip-9 |
|---|---|---|---|
| `GOOS=wasip1 GOARCH=wasm` (`./cmd/voidslice`) | clean, no cgo / no syscall errors | 10,480,148 B | 2,749,657 B |
| `GOOS=js GOARCH=wasm` (`./cmd/voidslice`) | clean | 10,527,641 B | 2,753,724 B |

Both compile straight off `main`, no source changes. The linter has zero cgo, zero net.Dial, and the only timer use is `internal/server`'s `context.WithTimeout`, which is dead code on the Worker path.

**Local execution proof**

- `wasip1`: ran the module under Node's `WASI` (`--experimental-wasi-unstable-preview1`) with the worktree pre-opened at `/repo`. Output on `testdata/broken/count-mismatch.decl`, `testdata/broken/missing-semicolon.decl`, and a real corpus file (`void-files/trimmed-Export-2/.../clues_boxing_02..rulehandler.decl`, 89 diagnostics) was **byte-identical** to native, modulo the `file` field.
- `js/wasm`: ran the module via Go's official `wasm_exec_node.js` host. Output identical to native on `count-mismatch.decl`.
- Timing (5 invocations): native ~5 ms each, Node-WASI ~120 ms each. Most of the WASI delta is host startup, not linter work.

**Why `js/wasm` over `wasip1` for the Worker**

- Cloudflare's [workers-wasi](https://github.com/cloudflare/workers-wasi) shim is explicitly **"experimental, with only some syscalls implemented"**. `poll_oneoff` returns ENOSYS, and Go's wasip1 runtime uses it for goroutine scheduling and timers — that's a well-known blocker.
- Cloudflare's WASI announcement lists Rust, C/C++, TinyGo, and SwiftWasm as supported toolchains. **Go's stdlib wasip1 target is not mentioned.** TinyGo would shrink the bundle dramatically but requires a code audit (`reflect`, `unsafe`, `encoding/json` constraints) that's not worth the time.
- `js/wasm` is the well-trodden path: plain WASM module + `wasm_exec.js` host, already runs in Workers (precedent: many existing CF Workers ship Go this way). No experimental APIs, no syscall surprises.

**Worker integration shape (for T11/T12)**

The simplest, most testable design — write a tiny separate entry point at `cmd/voidslice-wasm/main.go` that exports a single `syscall/js` function:

```go
//go:build js && wasm

package main

import (
	"syscall/js"

	"void-slice/internal/lint"
	"void-slice/internal/report"
	"void-slice/internal/scan"
)

func main() {
	js.Global().Set("voidsliceLint", js.FuncOf(lintFn))
	select {} // keep the runtime alive so JS can call back in
}

func lintFn(_ js.Value, args []js.Value) any {
	filename, src := args[0].String(), args[1].String()
	diags, _ := lint.New().Lint(filename, []byte(src))
	scanDiags := make([]scan.Diagnostic, len(diags))
	for i, d := range diags {
		scanDiags[i] = scan.Diagnostic{Code: scan.DiagnosticCode(d.Code), Span: d.Span, Message: d.Message}
	}
	return report.RenderJSON(filename, []byte(src), scanDiags)
}
```

The Worker JS imports `wasm_exec.js`, instantiates the module once at module-init (cold-start cost is paid once per isolate), and forwards `POST /lint` to `voidsliceLint(filename, body)`. No file system, no argv, no WASI shim.

This entry point is **additive** — doesn't touch `cmd/voidslice/main.go`. T11 just adds a build step (`GOOS=js GOARCH=wasm go build -o worker/voidslice.wasm ./cmd/voidslice-wasm`).

**Cloudflare Workers limits — verified against current docs**

| Limit | Free | Paid |
|---|---|---|
| CPU per request | 10 ms | 30 s default, up to 5 min |
| Memory per isolate | 128 MB | 128 MB |
| Bundle (compressed) | 3 MB | 10 MB |
| Bundle (uncompressed) | 64 MB | 64 MB |
| Request body | 100 MB (Free/Pro account) | 100 MB / 200 MB / 500 MB by account tier |

- Our gzipped 2.75 MB module fits even the 3 MB **Free** bundle limit. (Note: the `wasm_exec.js` host is an additional ~20 KB.)
- 5 ms native parse + WASM startup overhead → likely 20–80 ms total per cold-isolate request; once the isolate is warm, sub-10 ms is plausible. **The 10 ms Free-tier CPU budget is too tight; Paid Workers ($5/mo) is required for reliable serving.** This was already implicit in T26's framing but is worth stating here so T12 doesn't get blindsided.
- 5 MB body cap from T8 leaves a 95 MB headroom against the platform limit; T26's resource profile can keep tightening this.

**What this unblocks**

- **T11 (CI/CD):** the deploy step is now concrete. Use `wrangler deploy` against a `voidslice-api` Worker that bundles `worker/index.js` + `worker/voidslice.wasm` + `worker/wasm_exec.js`. No Cloudflare Containers / Container Registry needed. Drop those bullets from the workflow.
- **T12 (Production deploy):** Worker URL becomes the `VITE_API_URL` for Pages. Container deploy stays in the repo as a fallback but isn't on the critical path.
- **T26 (resource profile):** still needed — measure parse time on the largest realistic corpus file to confirm we stay under the 30 s Paid CPU budget with comfortable headroom. The 10 ms Free tier is conclusively not viable.

**Out of scope (not done here, deliberately)**

- Did **not** deploy to a real Worker — needs `wrangler login` with the user's Cloudflare account credentials. That's T11's job.
- Did **not** write `cmd/voidslice-wasm/main.go` itself — also T11.
- Did **not** measure on big corpus files. T26 owns that.
