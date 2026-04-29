#!/bin/bash
# T23/T31: auto git mv a kanban ticket when its **Status:** field changes.
#
# PostToolUse hook input arrives as JSON on stdin:
#   {"tool_name":"Edit"|"Write", "tool_input":{"file_path":"...", ...}, ...}
# CLAUDE_PROJECT_DIR is the repo root.

set -u

input=$(cat)
file_path=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')
[[ -n "$file_path" ]] || exit 0

# Resolve relative paths against the project root.
project_dir="${CLAUDE_PROJECT_DIR:-$PWD}"
[[ "$file_path" == /* ]] || file_path="$project_dir/$file_path"

# Only act on kanban/<col>/<version>/*.md  (e.g. kanban/todo/meta/T17-foo.md)
[[ "$file_path" =~ /kanban/[^/]+/[^/]+/[^/]+\.md$ ]] || exit 0
[[ -f "$file_path" ]] || exit 0

# Read new status from the (already-written) file.
status=$(grep -m1 '\*\*Status:\*\*' "$file_path" 2>/dev/null \
  | sed 's/.*\*\*Status:\*\*[[:space:]]*//' | tr -d '[:space:]')

case "$status" in
  todo|in-progress|done) ;;
  *) exit 0 ;;
esac

filename=$(basename "$file_path")
version=$(basename "$(dirname "$file_path")")
column=$(basename "$(dirname "$(dirname "$file_path")")")
kanban_dir=$(dirname "$(dirname "$(dirname "$file_path")")")

# Already in the right column → no-op.
[[ "$column" == "$status" ]] && exit 0

dest_dir="$kanban_dir/$status/$version"
dest_path="$dest_dir/$filename"

mkdir -p "$dest_dir"
cd "$project_dir" || exit 0

if git mv "$file_path" "$dest_path" 2>/dev/null; then
  echo "kanban: $column/$version/$filename → $status/$version/$filename"
else
  # Fall back to a plain mv if the file isn't tracked yet (e.g. brand-new ticket).
  mv "$file_path" "$dest_path" && \
    echo "kanban: $column/$version/$filename → $status/$version/$filename (untracked)"
fi
