#!/usr/bin/env bash
# goal-dispatch.sh — prepare a worktree on a goal branch for a Claude Code session.
#
# Refuses if the goal's Paths overlap an `active` or `ongoing` goal, if the
# goal is missing required fields, or if the worktree already exists.
#
# Does not spawn an agent and does not open a PR; the human takes those steps.

set -euo pipefail
shopt -s globstar nullglob

GOAL_ID="${1:-}"
if [ -z "$GOAL_ID" ]; then
  echo "usage: tools/goal-dispatch.sh M<N>" >&2
  exit 2
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

GOAL_FILE="kanban/goals/${GOAL_ID}.md"
if [ ! -f "$GOAL_FILE" ]; then
  echo "goal not found: $GOAL_FILE" >&2
  exit 1
fi

# --- field parsers -----------------------------------------------------------

# Strip leading "**Status:**" + spaces + trailing period.
parse_status() {
  awk '
    /^\*\*Status:\*\*/ {
      sub(/^\*\*Status:\*\*[[:space:]]*/, "")
      sub(/\.[[:space:]]*$/, "")
      sub(/[[:space:]]+$/, "")
      print
      exit
    }
  ' "$1"
}

# Print one path per line. Section ends at first blank line or next `**Field:`.
parse_paths() {
  awk '
    /^\*\*Paths:\*\*/ { in_paths=1; next }
    in_paths && /^- / { sub(/^- /, ""); print; next }
    in_paths && /^$/ { in_paths=0; next }
    in_paths        { in_paths=0 }
  ' "$1"
}

# Expand globs against the working tree, return concrete file paths (sorted, unique).
# Uses bash globstar so `**` matches zero or more path segments.
expand_to_files() {
  local g f
  local -a hits=()
  for g in "$@"; do
    # Unquoted to allow glob expansion. Patterns are repo-relative, no spaces.
    for f in $g; do
      [ -f "$f" ] && hits+=("$f")
    done
  done
  if [ "${#hits[@]}" -gt 0 ]; then
    printf '%s\n' "${hits[@]}" | sort -u
  fi
}

# --- validate the dispatched goal -------------------------------------------

GOAL_STATUS=$(parse_status "$GOAL_FILE")
if [ "$GOAL_STATUS" != "active" ]; then
  echo "refusing: ${GOAL_ID} status is '${GOAL_STATUS:-<missing>}', expected 'active'." >&2
  exit 1
fi

mapfile -t GOAL_PATHS < <(parse_paths "$GOAL_FILE")
if [ "${#GOAL_PATHS[@]}" -eq 0 ]; then
  echo "refusing: ${GOAL_ID} has no \`**Paths:**\` field. Add it (or re-run /goal-define for the goal)." >&2
  exit 1
fi

mapfile -t GOAL_FILES < <(expand_to_files "${GOAL_PATHS[@]}")

# --- overlap check against active/ongoing goals -----------------------------

overlaps_found=0
for other in kanban/goals/M*.md; do
  [ "$other" = "$GOAL_FILE" ] && continue
  other_status=$(parse_status "$other")
  case "$other_status" in
    active|ongoing) ;;
    *) continue ;;
  esac
  mapfile -t other_paths < <(parse_paths "$other")
  [ "${#other_paths[@]}" -eq 0 ] && continue
  mapfile -t other_files < <(expand_to_files "${other_paths[@]}")
  [ "${#other_files[@]}" -eq 0 ] && continue
  intersection=$(comm -12 \
    <(printf '%s\n' "${GOAL_FILES[@]:-}") \
    <(printf '%s\n' "${other_files[@]}"))
  if [ -n "$intersection" ]; then
    if [ "$overlaps_found" -eq 0 ]; then
      echo "refusing: ${GOAL_ID} Paths overlap an active/ongoing goal." >&2
      overlaps_found=1
    fi
    other_id=$(basename "$other" .md)
    while IFS= read -r f; do
      echo "  ${other_id} (${other_status}): ${f}" >&2
    done <<<"$intersection"
  fi
done
[ "$overlaps_found" -ne 0 ] && exit 1

# --- derive slug, branch, worktree path -------------------------------------

# Title line is `# M<N> — <title>`. Em-dash or hyphen tolerated.
TITLE=$(head -n1 "$GOAL_FILE" | sed -E 's/^#[[:space:]]*M[0-9]+[[:space:]]*[—–-][[:space:]]*//')
SLUG=$(printf '%s' "$TITLE" \
  | tr '[:upper:]' '[:lower:]' \
  | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')
if [ -z "$SLUG" ]; then
  echo "refusing: could not derive slug from ${GOAL_FILE} title." >&2
  exit 1
fi

BRANCH="goal/${GOAL_ID}-${SLUG}"
WORKTREE_PATH="$(dirname "$REPO_ROOT")/void-slice-${GOAL_ID}-${SLUG}"

if [ -e "$WORKTREE_PATH" ]; then
  echo "refusing: worktree already exists at ${WORKTREE_PATH}; resume there or remove it first." >&2
  exit 1
fi

# --- branch + worktree setup -------------------------------------------------

git fetch origin main --quiet

if git show-ref --verify --quiet "refs/heads/$BRANCH"; then
  : # local branch exists, reuse
elif git show-ref --verify --quiet "refs/remotes/origin/$BRANCH"; then
  git branch "$BRANCH" "origin/$BRANCH"
else
  git branch "$BRANCH" "origin/main"
fi

# Surface (don't auto-rebase) if behind origin/main.
if ! git merge-base --is-ancestor "origin/main" "$BRANCH" 2>/dev/null; then
  echo "warn: ${BRANCH} is behind origin/main; rebase manually before opening PRs." >&2
fi

git worktree add "$WORKTREE_PATH" "$BRANCH"

cat <<EOF

Worktree ready: ${WORKTREE_PATH}
Branch:        ${BRANCH}

Next:
  cd ${WORKTREE_PATH} && claude
  /implement-ticket ${GOAL_ID}.<n>
EOF
