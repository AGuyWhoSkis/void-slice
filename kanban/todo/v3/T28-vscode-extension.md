# T28 · VS Code Extension

**Status:** todo
**Version:** v3
**Size:** medium

## What

A minimal VS Code extension that registers `voidslice lsp` as the language server for `.decl`, `.entitydef`, and `.entities` files. Packaged as a `.vsix` for local install; no Marketplace publishing required.

## Scope

**Directory:** `voidslice-vscode/`

**Key files:**
- `package.json` — extension manifest: `contributes.languages` for `.decl`/`.entitydef`/`.entities`, `activationEvents`, `engines.vscode` version
- `src/extension.ts` — `activate()` starts a `LanguageClient` with `{command: "voidslice", args: ["lsp"]}` as the server executable
- `tsconfig.json`, `.vscodeignore`

**Toolchain:** Node.js + `vscode-languageclient` npm package + `@vscode/vsce` for packaging. TypeScript compile (`tsc`) before `vsce package`.

**Not in scope:** Marketplace publishing, auto-update, bundling with the Go binary, Windows path resolution.

**Assumption:** `voidslice` is on the user's PATH. Document this in the extension's README.

## Dependencies

T14 (LSP server binary must exist and be on PATH)

## Verification

```bash
cd voidslice-vscode
npm install
npm run compile   # tsc
vsce package      # produces voidslice-vscode-*.vsix

# Install locally:
code --install-extension voidslice-vscode-*.vsix

# Open a .decl file with known errors -> diagnostics appear in Problems panel and as red squiggles
```
