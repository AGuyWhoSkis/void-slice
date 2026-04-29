# voidslice — VS Code extension

Real-time diagnostics for Dishonored 2 / DOTO game files (`.decl`, `.entitydef`, `.entities`) inside VS Code, powered by the `voidslice lsp` language server.

## Prerequisite

The `voidslice` binary must be on your `PATH`. Build it from the repo root:

```bash
go build -o voidslice ./cmd/voidslice
# place it somewhere on PATH, e.g.:
sudo mv voidslice /usr/local/bin/
```

## Build & install

```bash
cd voidslice-vscode
npm install
npm run compile
npx @vscode/vsce package          # produces voidslice-vscode-<version>.vsix
code --install-extension voidslice-vscode-*.vsix
```

Open any `.decl` / `.entitydef` / `.entities` file — diagnostics appear in the Problems panel and as inline squiggles.
