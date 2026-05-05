#!/usr/bin/env bash
# nits-bootstrap.sh — set up the long-lived `nits` branch and sibling worktree.
#
# Idempotent. Capture, surface, and adoption flows live in M10.2+; this script
# just stands up the storage substrate so they have somewhere to write to.

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

WORKTREE_PATH="$(dirname "$REPO_ROOT")/void-slice-nits"
SEED_PATH="kanban/nits/README.md"
SEED_LINE="Pending nits live here. See CLAUDE.md § Nits."

# 1. Ensure origin/nits exists, forking origin/main if needed.
if ! git ls-remote --exit-code --heads origin nits >/dev/null 2>&1; then
  git fetch origin main --quiet
  git push origin "refs/remotes/origin/main:refs/heads/nits" --quiet
fi
git fetch origin nits --quiet

# 2. Ensure the seed file exists on origin/nits. Build the commit via plumbing
#    so it is authored from this repo's PWD (matters for sandboxed signing
#    environments where only the primary worktree is a registered source).
if ! git ls-tree origin/nits -- "$SEED_PATH" 2>/dev/null | grep -q .; then
  SEED_BLOB=$(printf '%s\n' "$SEED_LINE" | git hash-object -w --stdin)
  TMP_INDEX=$(mktemp)
  trap 'rm -f "$TMP_INDEX"' EXIT
  GIT_INDEX_FILE="$TMP_INDEX" git read-tree origin/nits
  GIT_INDEX_FILE="$TMP_INDEX" git update-index --add \
    --cacheinfo "100644,$SEED_BLOB,$SEED_PATH"
  TREE=$(GIT_INDEX_FILE="$TMP_INDEX" git write-tree)
  SIGN_FLAG=()
  if [ "$(git config --bool commit.gpgsign 2>/dev/null || echo false)" = "true" ]; then
    SIGN_FLAG=(-S)
  fi
  COMMIT=$(git commit-tree "${SIGN_FLAG[@]}" "$TREE" -p origin/nits -m "Seed kanban/nits/")
  git push origin "$COMMIT:refs/heads/nits" --quiet
  git fetch origin nits --quiet
fi

# 3. Ensure local `nits` branch tracks origin/nits.
if ! git show-ref --verify --quiet refs/heads/nits; then
  git branch nits origin/nits
fi

# 4. Ensure the worktree exists.
if git worktree list --porcelain \
  | awk -v p="$WORKTREE_PATH" '$1=="worktree" && $2==p {found=1} END{exit !found}'; then
  : # already registered
elif [ -e "$WORKTREE_PATH" ]; then
  echo "refusing: ${WORKTREE_PATH} exists but is not a registered worktree." >&2
  exit 1
else
  git worktree add "$WORKTREE_PATH" nits --quiet
fi

echo "nits worktree at ${WORKTREE_PATH}"
