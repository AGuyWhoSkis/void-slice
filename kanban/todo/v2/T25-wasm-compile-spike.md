# T25 · WASM Compile Spike

**Status:** todo
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
