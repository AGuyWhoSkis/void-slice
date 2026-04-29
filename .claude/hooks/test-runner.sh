#!/bin/bash
# T18/T31: auto-run go test ./... after editing .go source files.
#
# PostToolUse hook input arrives as JSON on stdin:
#   {"tool_name":"Edit"|"Write", "tool_input":{"file_path":"...", ...}, ...}
# CLAUDE_PROJECT_DIR is the repo root.

set -u

input=$(cat)
file_path=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')
[[ -n "$file_path" ]] || exit 0

# Only trigger on .go source files (skip testdata, markdown, generated files).
[[ "$file_path" == *.go ]] || exit 0

cd "${CLAUDE_PROJECT_DIR:-$PWD}" || exit 0
go test ./...
