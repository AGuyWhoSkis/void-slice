# T27 · LSP Subprocess Integration Tests

**Status:** todo
**Version:** v3
**Size:** small

## What

End-to-end integration tests for the LSP server that build the `voidslice` binary and drive it over real stdin/stdout. Exercises the full `voidslice lsp` path — transport framing, dispatch, lint engine, and diagnostic publication — as an actual editor client would.

## Scope

**File:** `cmd/voidslice/lsp_integration_test.go` (`package main_test`)

Reuses `binaryPath` from `TestMain` in `main_test.go` — the binary is already built before tests run.

**Session under test:**
```
initialize           -> recv InitializeResult (textDocumentSync == 1)
initialized          (notification, no response expected)
textDocument/didOpen  broken fixture -> recv publishDiagnostics (non-empty, PARSE_EXPECTED_SEMICOLON present)
textDocument/didChange fixed content -> recv publishDiagnostics (diagnostics: [])
shutdown             -> recv null result
exit                 -> process exits 0
```

**lspSession helper** (local to test file):
- `startLSP(t) *lspSession` — spawns `binaryPath lsp`, wires stdin/stdout pipes
- `sess.send(v any)` — marshal + Content-Length frame to stdin
- `sess.recv() map[string]json.RawMessage` — read one framed message from stdout

**Frame reader:** duplicate the `readMsg` logic locally (~15 lines) rather than exporting it from `internal/lsp`.

**Fixtures:**
- Broken: `testdata/broken/missing-semicolon.decl` (produces `PARSE_EXPECTED_SEMICOLON`)
- Fixed: minimal valid `.decl` inline (Version 1 / component / cpntTest with semicolons)

**Additional test:** `exit` without prior `shutdown` -> process exits 1 (LSP spec requirement).

## Dependencies

T14 (LSP server must be complete and building)

## Verification

```bash
go test ./cmd/voidslice/... -run TestLSP -v
```
