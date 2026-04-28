# T14 · LSP Server

**Status:** todo  
**Version:** v3  
**Size:** large

## What

A Language Server Protocol server that wraps `internal/lint` and speaks JSON-RPC 2.0 over stdio. Gives editors (VS Code, Neovim, etc.) real-time diagnostics while editing game files, with no additional transport plumbing — it's a new file that imports `lint`, not a rewrite.

## Scope

**Protocol:** LSP 3.17, JSON-RPC 2.0 over stdio

**Minimum viable capabilities:**
- `initialize` / `initialized` / `shutdown` / `exit` lifecycle
- `textDocument/didOpen` → run `lint.New().Lint()`, publish diagnostics
- `textDocument/didChange` (full sync) → re-lint, publish diagnostics
- `textDocument/didClose` → clear diagnostics
- `textDocument/publishDiagnostics` → send diagnostics to client

**Not in scope for v3:** code actions, completions, hover, formatting, incremental sync

**Package:** `internal/lsp` (handler logic) + `cmd/voidslice-lsp` or `voidslice lsp` subcommand

**LSP ↔ lint mapping:**
- `lint.Diagnostic.Severity` → `lsp.DiagnosticSeverity` (Error=1, Warning=2)
- `lint.Diagnostic.Span` → `lsp.Range` via `scan.LineIndex`
- `lint.Diagnostic.Code` → `lsp.Diagnostic.Code` (string variant)

**VS Code extension (stretch):** A minimal `.vsix` that sets `voidslice lsp` as the language server for `.decl`, `.entitydef`, `.entities` files.

## Dependencies

T4 (lint facade), T5 (CLI — confirms binary entry point pattern)  
Does NOT depend on any v2 ticket.

**Parallel dev note:** T8 (HTTP server) also modifies `cmd/voidslice/main.go` (adds `serve` subcommand). If developed in parallel on `v2-dev`, resolve at merge by keeping both `lsp` and `serve` cases.

## Verification

```bash
# test via direct stdio exchange
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}' | ./voidslice lsp

# or use a test framework like github.com/sourcegraph/jsonrpc2
go test ./internal/lsp/...

# integration: open a .decl file in VS Code with the extension installed → diagnostics appear in Problems panel
```
