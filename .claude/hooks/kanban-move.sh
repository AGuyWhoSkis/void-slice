#!/bin/bash
# T23: auto git mv ticket file when **Status:** field changes

# Extract file_path from tool input JSON
if [[ "$CLAUDE_TOOL_INPUT" =~ \"file_path\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
  file_path="${BASH_REMATCH[1]}"
else
  exit 0
fi

# Resolve relative paths
[[ "$file_path" == /* ]] || file_path="${CLAUDE_WORKING_DIR}/${file_path}"

# Only act on kanban/<col>/<version>/*.md  (e.g. kanban/todo/meta/T17-foo.md)
[[ "$file_path" =~ /kanban/[^/]+/[^/]+/[^/]+\.md$ ]] || exit 0

# Read new status from the (already-written) file
status=$(grep -m1 '\*\*Status:\*\*' "$file_path" 2>/dev/null \
  | sed 's/.*\*\*Status:\*\*[[:space:]]*//' | tr -d '[:space:]')

case "$status" in
  todo|in-progress|done) ;;
  *) exit 0 ;;
esac

# Parse current path components
filename=$(basename "$file_path")
version=$(basename "$(dirname "$file_path")")
column=$(basename "$(dirname "$(dirname "$file_path")")")
kanban_dir=$(dirname "$(dirname "$(dirname "$file_path")")")

# Nothing to do if already in the right column
[[ "$column" == "$status" ]] && exit 0

dest_dir="$kanban_dir/$status/$version"
mkdir -p "$dest_dir"
cd "${CLAUDE_WORKING_DIR:-.}"
git mv "$file_path" "$dest_dir/$filename"
echo "kanban: $column/$version/$filename → $status/$version/$filename"
