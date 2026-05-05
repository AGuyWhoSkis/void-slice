#!/usr/bin/env bash
# nit-claim.sh — atomically claim a pending nit for a goal.
#
# Builds a delete-the-nit-file commit on top of origin/nits via plumbing and
# pushes it. The push is the atomic claim: if a concurrent slice claimed the
# same nit first, the push is non-fast-forward and we exit 2.
#
# Args: <goal-id> <nit-filename>     (filename is just the basename, no path)
# Exit:  0 win
#        1 misuse / unexpected error
#        2 race lost (non-fast-forward)
#        3 network failure after retries

set -euo pipefail

GOAL_ID="${1:-}"
NIT_FILE="${2:-}"
if [ -z "$GOAL_ID" ] || [ -z "$NIT_FILE" ]; then
  echo "usage: tools/nit-claim.sh <goal-id> <nit-filename>" >&2
  exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

WORKTREE_PATH="$(dirname "$REPO_ROOT")/void-slice-nits"
if [ ! -d "$WORKTREE_PATH" ]; then
  echo "nits worktree not found at ${WORKTREE_PATH} — run tools/nits-bootstrap.sh first" >&2
  exit 1
fi

NIT_REL="kanban/nits/${NIT_FILE}"

# Sync the worktree and build the claim commit on top of origin/nits.
git fetch origin nits --quiet
git -C "$WORKTREE_PATH" reset --hard origin/nits --quiet

if ! git ls-tree origin/nits -- "$NIT_REL" 2>/dev/null | grep -q .; then
  echo "claimed by another commit on origin/nits" >&2
  exit 2
fi

TMP_INDEX=$(mktemp)
trap 'rm -f "$TMP_INDEX"' EXIT
GIT_INDEX_FILE="$TMP_INDEX" git read-tree origin/nits
GIT_INDEX_FILE="$TMP_INDEX" git update-index --remove "$NIT_REL"
TREE=$(GIT_INDEX_FILE="$TMP_INDEX" git write-tree)

SIGN_FLAG=()
if [ "$(git config --bool commit.gpgsign 2>/dev/null || echo false)" = "true" ]; then
  SIGN_FLAG=(-S)
fi
COMMIT=$(git commit-tree "${SIGN_FLAG[@]}" "$TREE" -p origin/nits \
  -m "claim ${NIT_FILE} for ${GOAL_ID}")

# Push with backoff, distinguishing race-lost (non-FF) from network failure.
# We push the new commit pinned to origin/nits's expected tip via --force-with-
# lease=nits:<sha> so a concurrent claim races us cleanly.
EXPECTED=$(git rev-parse origin/nits)
delays=(2 4 8 16)
attempt=0
while : ; do
  if push_err=$(git push origin "$COMMIT:refs/heads/nits" \
    --force-with-lease="refs/heads/nits:$EXPECTED" --quiet 2>&1); then
    break
  fi
  case "$push_err" in
    *"non-fast-forward"*|*"stale info"*|*"force-with-lease"*|*"rejected"*)
      echo "claimed by another commit on origin/nits" >&2
      exit 2
      ;;
  esac
  if [ "$attempt" -ge "${#delays[@]}" ]; then
    echo "push failed after retries: $push_err" >&2
    exit 3
  fi
  sleep "${delays[$attempt]}"
  attempt=$((attempt + 1))
done

git fetch origin nits --quiet
git -C "$WORKTREE_PATH" reset --hard origin/nits --quiet

echo "claimed: ${NIT_FILE}"
