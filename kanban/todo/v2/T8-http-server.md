# T8 · HTTP Server (`internal/server`)

**Status:** todo  
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
- Max body: 5MB — reject with `413` if exceeded
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
