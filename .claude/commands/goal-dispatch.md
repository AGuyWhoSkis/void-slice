Prepare a worktree on the goal branch for $ARGUMENTS so an agent session can start work. `$ARGUMENTS` is a goal-id like `M9`.

This command **refuses** when:
- The goal file doesn't exist.
- The goal's `**Status:**` isn't `active`.
- The goal has no `**Paths:**` field.
- The goal's `Paths:` overlap (set-intersection of matched files) those of another `active` or `ongoing` goal.
- The target worktree path already exists.

Otherwise it:
1. Derives `<slug>` from the goal title (lowercase, hyphenate, drop punctuation) and uses branch `goal/M<N>-<slug>`.
2. Reuses the branch if it exists locally or on `origin`; otherwise creates it from `origin/main`.
3. Warns (does not auto-rebase) if the goal branch is behind `origin/main`.
4. Adds a worktree at `../void-slice-M<N>-<slug>` (sibling of the repo root).
5. Prints next-step instructions: `cd <worktree> && claude`, then `/implement-ticket M<N>.<n>`.

This command does **not** spawn an agent and does **not** open a PR. The first `git push` from the goal branch surfaces GitHub's PR-create URL, or run `gh pr create` manually.

## Run

```sh
tools/goal-dispatch.sh $ARGUMENTS
```

Report the script's stdout/stderr to the user verbatim. On non-zero exit, echo the error and stop — do not attempt to recover.
