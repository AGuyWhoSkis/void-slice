# T-v3-followups · v3 Follow-up Tickets (Upcoming)

**Status:** todo
**Version:** stretch (evaluate after v2 lands)
**Size:** umbrella — split into 4 child tickets when picked up

## What

Tracking ticket for gaps observed at the close of v3. v3 shipped a working LSP server, subprocess integration tests, and a buildable VS Code extension — sufficient for a developer running VS Code from this repo. The items below close the gap between "developer can use it" and "any modder can install it on any LSP-capable editor and we'll know if it breaks."

Promote each child to its own ticket (numbered T29+) when scheduled.

## Children

### T29 — LSP client setup docs (Neovim, Helix, Zed) — small

T14's stated audience was "VS Code, Neovim, etc." but only VS Code got a client. Add `docs/editor-setup.md` (or a top-level README section) with copy-pasteable snippets:

- `nvim-lspconfig` config registering `voidslice lsp` for the three filetypes
- `helix languages.toml` block
- `zed` extensions / settings note

No code, just docs. Closes the scope gap from T14.

### T30 — VS Code extension end-to-end smoke test — medium

`extension.ts` is currently unverified by automation. If someone breaks the activation event, document selector, or the `LanguageClient` wiring, every Go test still passes.

- Add `@vscode/test-electron` runner under `voidslice-vscode/`
- Test: launch real VS Code, open `testdata/broken/missing-semicolon.decl`, assert `vscode.languages.getDiagnostics()` returns a non-empty list including `PARSE_EXPECTED_SEMICOLON`
- CI: needs xvfb on Linux runners

Only test that catches extension-side regressions.

### T31 — LSP server stderr logging — small

The server is opaque: no way for a user to tell why diagnostics aren't appearing.

- Log one line per inbound method (`<- initialize`, `<- textDocument/didOpen file:///x.decl`)
- Log one line per outbound publishDiagnostics (`-> publishDiagnostics file:///x.decl 3 diags`)
- Gate on `--verbose` flag or `VOIDSLICE_LSP_LOG=1` env var so production stderr stays quiet
- Stderr only — never stdout (would corrupt the JSON-RPC stream)

Cheap; high payoff first time something goes wrong in the wild.

### T32 — Release pipeline for the binary + .vsix — medium

Non-Go modders currently have no install path: no released binary, no published `.vsix`. On tag push:

- Cross-build `voidslice` for linux/macOS/windows
- Run `npx @vscode/vsce package`
- Attach all artifacts to a GitHub Release

Does **not** publish to the VS Code Marketplace (out of scope per T28). Pairs with v2 T11 (CI/CD) — likely shares the workflow file.

## Code-cleanup observations (not ticket-worthy on their own; fold into the next LSP work)

- `Server.docs[uri]` is written by `handleDidOpen` / `handleDidChange` but never read. Either remove the field or earmark it for the first feature that needs it (hover, go-to-definition). Lean toward delete-now.
- `convert.spanToRange` uses byte offsets for LSP `Position.Character`; the spec says UTF-16 code units. Game files are ASCII so the highlight is invisibly correct; fix when a non-ASCII file shows up.
- `dispatch` silently drops malformed requests instead of returning JSON-RPC `-32700` (parse error) or `-32602` (invalid params). Theoretical compliance debt — real LSP clients don't send malformed requests.

## Dependencies

T14, T27, T28 (all done).
T32 lines up well with v2 T11 — schedule together if v2 still in flight.

## Verification

Each child ticket carries its own verification. This umbrella ticket is "done" once all four children are filed (or explicitly declined) — it has no implementation of its own.
