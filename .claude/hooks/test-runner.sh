#!/bin/bash
# T18: auto-run go test ./... after editing .go source files
# To also run go vet, change the last line to: go test ./... && go vet ./...

# Extract file_path from tool input JSON
if [[ "$CLAUDE_TOOL_INPUT" =~ \"file_path\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
  file_path="${BASH_REMATCH[1]}"
else
  exit 0
fi

# Only trigger on .go source files (skip testdata, markdown, generated files)
[[ "$file_path" == *.go ]] || exit 0

cd "${CLAUDE_WORKING_DIR:-.}"
go test ./...
