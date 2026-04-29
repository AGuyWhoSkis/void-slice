# T28 · VS Code Extension

**Status:** done
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

## Completion

Created `voidslice-vscode/` with manifest, TypeScript source, build config, and README:

- `package.json` — declares three languages (`voidslice-decl` / `voidslice-entitydef` / `voidslice-entities`) bound to `.decl` / `.entitydef` / `.entities`, activates on those languages, and depends on `vscode-languageclient ^9`.
- `src/extension.ts` — `activate()` constructs a `LanguageClient` with `serverOptions = {command: "voidslice", args: ["lsp"], transport: stdio}` for both run and debug, and a document selector matching the three languages. `deactivate()` returns `client.stop()`.
- `tsconfig.json`, `.vscodeignore`, `.gitignore` (excludes `node_modules/`, `out/`, `*.vsix`, `package-lock.json`), `README.md` (install/usage notes; documents that `voidslice` must be on PATH).

Toolchain decisions:
- Used `npx @vscode/vsce` (declared as a devDependency) rather than installing `vsce` globally.
- Did not commit `package-lock.json` — for an extension this size the determinism gain isn't worth the churn.

Verification (in `voidslice-vscode/`):
- `npm install` -> 315 packages, 0 vulnerabilities.
- `npm run compile` (= `tsc -p ./`) -> emits `out/extension.js`, no errors.
- `npx @vscode/vsce package` -> produces `voidslice-vscode-0.1.0.vsix` (2.7 KB, 5 files). Two non-fatal warnings about a missing `repository` field and `LICENSE` file — left as-is since this extension isn't being published to the Marketplace.

Out of scope per the ticket: actual VS Code install (`code --install-extension`) and end-to-end "open a .decl, see squiggles" verification — those require an interactive VS Code session and are documented in the README for the user to run.
