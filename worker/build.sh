#!/bin/bash
# Build the Worker bundle: voidslice.wasm + wasm_exec.js shim from the active
# Go toolchain. Run from the repo root.
#
# Usage: ./worker/build.sh
# Outputs: worker/voidslice.wasm, worker/wasm_exec.js (both gitignored).

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$script_dir/.." && pwd)
cd "$repo_root"

echo "+ building voidslice.wasm (GOOS=js GOARCH=wasm)"
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" \
  -o worker/voidslice.wasm ./cmd/voidslice-wasm

goroot=$(go env GOROOT)
shim="$goroot/lib/wasm/wasm_exec.js"
if [[ ! -f "$shim" ]]; then
  shim="$goroot/misc/wasm/wasm_exec.js"
fi
if [[ ! -f "$shim" ]]; then
  echo "wasm_exec.js not found under $goroot" >&2
  exit 1
fi
cp "$shim" worker/wasm_exec.js

raw=$(stat -c%s worker/voidslice.wasm)
gz=$(gzip -9 -c worker/voidslice.wasm | wc -c)
echo "+ voidslice.wasm: ${raw} bytes raw, ${gz} bytes gzipped"
