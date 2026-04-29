# T8 · HTTP Server (`internal/server`)

**Status:** done  
**Version:** v2  
**Size:** medium  
**Blocks:** T9 (frontend)

## What

Implement `internal/server` — a minimal HTTP server that exposes `POST /lint` using the same `internal/lint` engine as the CLI. This is the backend the React playground talks to.

## Scope

**Package:** `internal/server`

**Endpoint: `POST /lint`**
- Request body: raw file bytes (multipart or `application/octet-stream`)
- Response: JSON array of `lint.Diagnostic` (same structure as `report.RenderJSON`)
- Include `"file"` field derived from the uploaded filename or a default

**Input limits:**
- Max body: 5MB — reject with `413` if exceeded (provisional; T26 resource profile may adjust this)
- Parse timeout: 5s — use `context.WithTimeout`, respond `504` if exceeded
- Content-type validation: reject anything clearly non-text without binary-sniff fallback

**CORS:**
- Allow the frontend origin (Cloudflare Pages domain). Use an `ALLOWED_ORIGIN` env var with a sensible default (`"*"` for local dev, locked down in prod)

**Health check:**
- `GET /health` → `200 OK` with `{"status":"ok"}`

**Structured logging:**
- Use `log/slog` (stdlib, Go 1.21+)
- Log: method, path, status, duration, file size per request

**Server entry point:**
- `cmd/voidslice/main.go` gains a `serve` subcommand: `voidslice serve [--port 8080]`
- Or a separate `cmd/server/main.go` — decide at implementation time based on binary size preference

## Dependencies

T4 (lint facade), T5 (CLI — confirms the binary structure is settled)

**Parallel dev note:** T14 (LSP server) also modifies `cmd/voidslice/main.go` (adds `lsp` subcommand). If developed in parallel on `v3-dev`, resolve at merge by keeping both `serve` and `lsp` cases.

## Verification

```bash
go build ./...
./voidslice serve --port 8080 &

# health
curl http://localhost:8080/health

# lint a file
curl -X POST http://localhost:8080/lint \
  --data-binary @testdata/broken/count-mismatch.decl \
  -H "Content-Type: application/octet-stream"
# expect: JSON array with diagnostics

# oversized input
dd if=/dev/zero bs=1M count=6 | curl -X POST http://localhost:8080/lint --data-binary @-
# expect: 413

go test ./internal/server/...
```

## Completion

**What was done**

- New package `internal/server` with `Config`, `New(http.Handler)`, `ListenAndServe`. Handler exposes `GET /health`, `POST /lint`, and `OPTIONS /lint` preflight; everything else on `/lint` returns 405.
- `POST /lint` accepts `application/octet-stream`, `text/*`, and `multipart/form-data` bodies. Multipart filenames flow through to the response `file` field; octet-stream callers can pass `?filename=…`. Defaults to `"input"`.
- Body size enforcement via `http.MaxBytesReader` (default 5 MiB). `*http.MaxBytesError` is mapped to 413 for both the streaming and multipart paths.
- Lint runs in a goroutine with `context.WithTimeout` (default 5 s). On timeout the response is 504; the goroutine drains via a buffered channel.
- Content-Type allow-list rejects `image/*`, `audio/*`, `video/*`, and a small set of archive/PDF MIME types with 415. Unknown types are accepted (the lint engine still binary-sniffs).
- CORS middleware sets `Allow-Origin / Allow-Methods / Allow-Headers` on every response and adds `Vary: Origin` when origin is non-`*`.
- Per-request slog line: method, path, status, duration_ms, bytes_in.
- `cmd/voidslice/main.go` refactored into a subcommand dispatcher (`lint`, `serve`). `runServe` reads `ALLOWED_ORIGIN`, builds a JSON slog handler on stderr, and calls `server.ListenAndServe`.
- 10 table-driven tests in `internal/server/server_test.go` cover health, broken/clean lints, 413, 415, multipart filename pass-through, 504 via injected timeout, OPTIONS preflight, 405 on GET, and the success-path CORS header.

**Key decisions**

- Single-binary `voidslice serve` rather than a separate `cmd/server`, matching T10's docker-compose `command: ["serve", "--port", "8080"]`.
- Reused `report.RenderJSON` for the response body so the wire format stays identical to the CLI's `--json` output. The 6-line `lint.Diagnostic → scan.Diagnostic` adapter is duplicated between `cmd/voidslice` and `internal/server` — two callers is below the threshold for a shared helper.
- Env coupling (`ALLOWED_ORIGIN`) lives in `main.go`, not the server package, so tests don't need env stubs.

**Deviations**

- None from the ticket scope.

**Verification**

- `go vet ./...` clean.
- `go test ./...` green (after symlinking `void-files/` into the worktree — gitignored corpus, pre-existing scan-test dependency).
- Manual smoke confirmed: `/health` → `{"status":"ok"}`; `/lint` on `count-mismatch.decl` returns the expected `VALIDATE_ARRAY_COUNT_MISMATCH` warning; 6 MiB body → 413; OPTIONS → 204 with CORS headers.
