# T16 · Git Worktrees for Parallel Claude Sessions

**Status:** done  
**Version:** meta  
**Size:** small

## What

Use `git worktree add` (or `claude --worktree <branch>` if the built-in handles setup/teardown) to run two or more Claude Code sessions simultaneously on separate branches without branch-switching friction. Each worktree is an independent directory sharing the same git object store.

## Scope

- Establish the standard invocation: `git worktree add ../void-slice-<branch> <branch>`
- Document where worktree directories should land relative to the repo root
- Note cleanup: `git worktree remove <path>` and `git worktree prune`
- Determine whether `claude --worktree` supersedes manual `git worktree add`; prefer the built-in if it handles setup and teardown automatically
- Confirm `.claude/` settings are inherited correctly in each worktree

## Dependencies

None (no prerequisites)

**Downstream:** T22 requires this ticket — do T16 before running the subagents trial. Once the worktree convention is settled, document it in T20's **Tooling** section.

## Verification

```bash
git worktree add ../void-slice-feature some-branch
# open a second Claude Code session in ../void-slice-feature
# confirm edits in one session don't appear in the other
git worktree remove ../void-slice-feature
```

## Completion

Convention established and documented in `CLAUDE.md § Tooling`:

- **Preferred invocation:** `claude --worktree <branch>` — Claude Code manages setup and teardown automatically
- **Manual fallback:** `git worktree add ../void-slice-<branch> <branch>` — worktrees land as siblings to the repo root
- **`.claude/` inheritance:** Settings are tracked in git; each worktree's branch has its own copy of `.claude/settings.json`
- **Cleanup:** `git worktree remove ../void-slice-<branch>` then `git worktree prune`

No code changes. Convention lives in `CLAUDE.md`.
