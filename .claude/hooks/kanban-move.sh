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

# Only act on kanban/<col>/*.md  (e.g. kanban/todo/M1.10-cli-polish.md).
# Explicitly does NOT match kanban/goals/*.md — goals don't auto-move.
[[ "$file_path" =~ /kanban/(todo|in-progress|done)/[^/]+\.md$ ]] || exit 0
[[ -f "$file_path" ]] || exit 0

# Read new status from the (already-written) file.
status=$(grep -m1 '\*\*Status:\*\*' "$file_path" 2>/dev/null \
  | sed 's/.*\*\*Status:\*\*[[:space:]]*//' | tr -d '[:space:]')

case "$status" in
  todo|in-progress|done) ;;
  *) exit 0 ;;
esac

filename=$(basename "$file_path")
column=$(basename "$(dirname "$file_path")")
kanban_dir=$(dirname "$(dirname "$file_path")")

# Already in the right column → no-op.
[[ "$column" == "$status" ]] && exit 0

dest_dir="$kanban_dir/$status"
dest_path="$dest_dir/$filename"

mkdir -p "$dest_dir"
cd "$project_dir" || exit 0

if git mv "$file_path" "$dest_path" 2>/dev/null; then
  echo "kanban: $column/$filename → $status/$filename"
else
  # Fall back to a plain mv if the file isn't tracked yet (e.g. brand-new ticket).
  mv "$file_path" "$dest_path" && \
    echo "kanban: $column/$filename → $status/$filename (untracked)"
fi
