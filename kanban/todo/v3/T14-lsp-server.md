# T14 · LSP Server

**Status:** todo
**Version:** v3
**Size:** large

## What

A Language Server Protocol server that wraps `internal/lint` and speaks JSON-RPC 2.0 over stdio. Gives editors (VS Code, Neovim, etc.) real-time diagnostics while editing game files. Zero new external dependencies — hand-rolled Content-Length framing + hand-written LSP wire structs.

## Scope

**Protocol:** LSP 3.17, JSON-RPC 2.0 over stdio

**Package layout:**
```
internal/lsp/
    types.go      — hand-written LSP wire structs (Position, Range, Diagnostic, etc.)
    transport.go  — readMsg(*bufio.Reader), writeMsg(io.Writer, any)
    server.go     — Server struct, New(), Serve(r, w), processMessage()
    dispatch.go   — dispatch(), per-method handlers, publishDiagnostics()
    convert.go    — spanToRange(), convertDiagnostics(); scan.Pos (1-based) -> LSP Position (0-based)
    lsp_test.go   — unit tests (package lsp, mock io.Writer)
```

**Subcommand:** `voidslice lsp` — refactor `main.go` to a `switch os.Args[1]` dispatching `runLint` and `runLSP`.

**Methods handled:**

| Method | id? | Action |
|--------|-----|--------|
| `initialize` | yes | respond `{capabilities:{textDocumentSync:1}}` |
| `initialized` | no | no-op |
| `shutdown` | yes | set shutdownRequested; respond null |
| `exit` | no | `os.Exit(0)` if shutdown was called, else `os.Exit(1)` |
| `textDocument/didOpen` | no | cache content, lint, publishDiagnostics |
| `textDocument/didChange` | no | update cache (full sync), lint, publishDiagnostics |
| `textDocument/didClose` | no | delete from cache, publishDiagnostics `[]` |
| unknown request | yes | error -32601 |
| unknown notification | no | silently ignore |

**Not in scope:** code actions, completions, hover, incremental sync, VS Code extension (T28)

**Key invariants:**
- `publishDiagnostics` sends `"diagnostics":[]` (not null) for clean files
- `rpcEnvelope.ID *json.RawMessage` — nil = notification, non-nil = request (must respond)
- `bufio.Reader` created once in `Serve`, reused across all `readMsg` calls

## Dependencies

T4 (lint facade), T5 (CLI entry point pattern)
Does NOT depend on any v2 ticket.
T27 (integration tests) depends on this ticket.

**Merge note:** T8 (v2, HTTP server) also modifies `cmd/voidslice/main.go`. Resolve by keeping both `serve` and `lsp` cases.

## Verification

```bash
go test ./internal/lsp/...   # unit tests pass
go vet ./internal/lsp/...
go build ./cmd/voidslice/...

# smoke test -- expect Content-Length framed InitializeResult:
printf 'Content-Length: 77\r\n\r\n{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' \
  | ./voidslice lsp
```

**Unit test cases (lsp_test.go):**
1. `initialize` -> `result.capabilities.textDocumentSync == 1`
2. `textDocument/didOpen` with broken fixture -> `publishDiagnostics`, codes include `PARSE_EXPECTED_SEMICOLON`
3. `textDocument/didChange` with fixed content -> `publishDiagnostics` with `diagnostics: []`
4. `shutdown` -> null result, `shutdownRequested == true`; unknown notification -> no output
