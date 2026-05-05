#!/usr/bin/env bash
# nit.sh — file a nit on the long-lived `nits` branch.
#
# Writes a markdown file at kanban/nits/<timestamp>-<slug>.md in the sibling
# `../void-slice-nits/` worktree, then commits and pushes. The commit is built
# via plumbing from the primary worktree's PWD so signing works in sandboxes
# where only the primary worktree is a registered code-signing source.

set -euo pipefail

DESCRIPTION=""
PATHS=""
CONTEXT=""

while [ $# -gt 0 ]; do
  case "$1" in
    --description) DESCRIPTION="$2"; shift 2 ;;
    --paths)       PATHS="$2";       shift 2 ;;
    --context)     CONTEXT="$2";     shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [ -z "$DESCRIPTION" ] || [ -z "$PATHS" ]; then
  echo "usage: tools/nit.sh --description <str> --paths <glob> [--context <str>]" >&2
  exit 2
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
ORIGIN_BRANCH=$(git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD)
cd "$REPO_ROOT"

WORKTREE_PATH="$(dirname "$REPO_ROOT")/void-slice-nits"
if [ ! -d "$WORKTREE_PATH" ]; then
  echo "nits worktree not found at ${WORKTREE_PATH} — run tools/nits-bootstrap.sh first" >&2
  exit 1
fi

# Sync the worktree to origin/nits before writing.
git fetch origin nits --quiet
git -C "$WORKTREE_PATH" reset --hard origin/nits --quiet

TS=$(date -u +%Y%m%dT%H%M%SZ)

# Slug: lowercase, [a-z0-9-] only, collapsed hyphens, ≤40 chars, no trailing -.
SLUG=$(printf '%s' "$DESCRIPTION" \
  | tr '[:upper:]' '[:lower:]' \
  | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' \
  | cut -c1-40 \
  | sed -E 's/-+$//')
if [ -z "$SLUG" ]; then
  echo "could not derive slug from description" >&2
  exit 1
fi

NIT_REL="kanban/nits/${TS}-${SLUG}.md"
NIT_ABS="$WORKTREE_PATH/$NIT_REL"

mkdir -p "$(dirname "$NIT_ABS")"
{
  printf '# %s\n\n' "$DESCRIPTION"
  printf '**Captured:** %s\n' "$TS"
  printf '**Branch:** %s\n' "$ORIGIN_BRANCH"
  printf '**Paths:**\n- %s\n' "$PATHS"
  if [ -n "$CONTEXT" ]; then
    printf '\n%s\n' "$CONTEXT"
  fi
} >"$NIT_ABS"

# Build the commit via plumbing (signed if the repo is configured to sign).
BLOB=$(git hash-object -w --stdin <"$NIT_ABS")
TMP_INDEX=$(mktemp)
trap 'rm -f "$TMP_INDEX"' EXIT
GIT_INDEX_FILE="$TMP_INDEX" git read-tree origin/nits
GIT_INDEX_FILE="$TMP_INDEX" git update-index --add \
  --cacheinfo "100644,$BLOB,$NIT_REL"
TREE=$(GIT_INDEX_FILE="$TMP_INDEX" git write-tree)

SIGN_FLAG=()
if [ "$(git config --bool commit.gpgsign 2>/dev/null || echo false)" = "true" ]; then
  SIGN_FLAG=(-S)
fi
COMMIT=$(git commit-tree "${SIGN_FLAG[@]}" "$TREE" -p origin/nits \
  -m "nit: ${DESCRIPTION}")

# Push with exponential backoff on network failure.
delays=(2 4 8 16)
attempt=0
while : ; do
  if git push origin "$COMMIT:refs/heads/nits" --quiet; then
    break
  fi
  if [ "$attempt" -ge "${#delays[@]}" ]; then
    echo "captured locally on nits branch but not pushed; run 'git -C ${WORKTREE_PATH} fetch origin && git -C ${WORKTREE_PATH} push origin nits' when network returns" >&2
    # Leave the worktree pointing at the new commit so a manual push works.
    git -C "$WORKTREE_PATH" update-ref refs/heads/nits "$COMMIT"
    git -C "$WORKTREE_PATH" reset --hard nits --quiet
    exit 1
  fi
  sleep "${delays[$attempt]}"
  attempt=$((attempt + 1))
done

# Re-sync the worktree to the pushed commit.
git fetch origin nits --quiet
git -C "$WORKTREE_PATH" reset --hard origin/nits --quiet

echo "filed: ${NIT_REL}"
