# Kanban Board

## Goals

Goals capture durable intent — the why and the scope of a body of work. Tickets are disposable execution units that reference a goal by ID-prefix in their filename.

| Goal | File |
|------|------|
| [M1 — Linter engine + CLI](goals/M1.md) | `goals/M1.md` |
| [M2 — Cloud playground](goals/M2.md) | `goals/M2.md` |
| [M3 — Editor integration (LSP)](goals/M3.md) | `goals/M3.md` |
| [M4 — Linter scope refinement](goals/M4.md) | `goals/M4.md` |
| [M5 — Dev tooling & workflow](goals/M5.md) | `goals/M5.md` |
| [M6 — Preview environments](goals/M6.md) | `goals/M6.md` |
| [M7 — Production hardening](goals/M7.md) | `goals/M7.md` |
| [M8 — Playground bisection ladder](goals/M8.md) | `goals/M8.md` |
| [M9 — Concurrent agent workflow](goals/M9.md) | `goals/M9.md` |
| [M10 — Nit capture](goals/M10.md) | `goals/M10.md` |
| [M12 — Lint engine hardening](goals/M12.md) | `goals/M12.md` |

### `Paths:` field

Active goals declare a `**Paths:**` block between `**Scope:**` and `**Status:**` — a bullet list of repo-relative globs naming the surface the goal will edit. Lets the dispatcher refuse to spawn two agents whose surfaces overlap, without parsing prose.

```markdown
**Paths:**
- internal/scan/**
- internal/parse/**
```

- **Glob syntax.** Doublestar globs (`**`, `*`, `?`), repo-relative, no leading slash. Same conventions as Go tooling (`golangci-lint`, etc.).
- **Overlap.** Two `Paths:` fields overlap iff some concrete file in the working tree matches a pattern in both — set-intersection of matched paths, not pattern-prefix or shared-directory.
- **Cross-cutting files always declared.** No implicit allowlist for things like `go.mod`, `Makefile`, `CLAUDE.md`, `kanban/README.md`. If two goals both edit `Makefile`, they overlap and must serialize.
- **Closed goals stay bare.** Overlap checking only matters for in-flight goals; M1–M8 don't carry `Paths:` retroactively.

## Nits

A nit is a tiny single-edit chore captured during another goal's work — see [CLAUDE.md § Nits](../CLAUDE.md#nits) for the rationale and capture/surface/adopt flow.

Substrate: a long-lived `nits` branch with a sibling worktree at `../void-slice-nits/`, bootstrapped via [`tools/nits-bootstrap.sh`](../tools/nits-bootstrap.sh). Pending nits live on that branch under `kanban/nits/`. Adopted nits are removed from the `nits` branch (the `git push` that removes them is the atomic claim) and re-emerge on the adopting goal branch as ordinary tickets carrying `**Origin:** nit-<timestamp>`.

Lifecycle: `filed → claimed → ticketed → landed`. Closing the materialized ticket follows the standard ticket lifecycle.

Sample nit file:

```markdown
# never include claude.ai/code/session_* URLs in commit messages or PR bodies

**Captured:** 20260505T143022Z
**Branch:** goal/M10-nit-capture
**Paths:**
- CLAUDE.md

Claude Code's git templates embed `claude.ai/code/session_<id>` URLs by default. Even though they're auth-gated, git history is permanent and access models can change — opt out repo-wide.
```

## Columns

| Folder | Meaning |
|--------|---------|
| `todo/` | Prioritized and ready to start |
| `in-progress/` | Being worked on right now |

Status folders are flat. Goal-membership is encoded in the filename prefix. Closed tickets are deleted (`git rm`) — there is no `done` folder or status; `git log` is the closure record.

## Ticket format

Filename: `<goal-id>.<n>-short-name.md`. Examples: `M1.10-cli-polish.md`, `M5.7-subagents-eval.md`, `M3.4-lsp-followups.md`.

Edit the `**Status:**` field (`todo` / `in-progress`) — the kanban-move hook moves the file into the matching folder. Do not rename or `git mv` manually.

New tickets pick the next free `<n>` within their goal:

```
ls kanban/{todo,in-progress}/<goal-id>.* 2>/dev/null | wc -l
```

Add 1 to that count.

## Lifecycle

1. **Create** — write `kanban/todo/M<N>.<n>-name.md` with `**Status:** todo`. No other file is touched.
2. **Pick up** — edit `**Status:** in-progress`. The hook moves the file to `kanban/in-progress/`.
3. **Close** — `git rm` the ticket file. The commit message is the closure record; optionally append a brief retro to the parent goal file. There is no `done` folder or status.

Closing a goal is just flipping its `**Status:**` field. A retrospective on the goal file is optional, not mandatory; [goals/M2.md](goals/M2.md) and [goals/M6.md](goals/M6.md) serve as exemplars.

### Goal branches

Each active goal owns a long-lived branch `goal/M<N>-<slug>`, where `<slug>` is the goal file's title lowercased and hyphenated (e.g. M9 "Concurrent agent workflow" → `goal/M9-concurrent-agent-workflow`). The slug is deterministic so `/goal-dispatch` can reproduce the branch name from a goal-id alone.

The branch is created from `main` (manually or by `/goal-dispatch`). Tickets for the goal commit onto it; `/implement-ticket` refuses to run if the working tree isn't on the matching goal branch. A single PR per goal targets `main` and is opened early (the slice itself is a deliverable). The PR is merged by a human when the goal closes — branch protection on `main` enforces the gate.

`/goal-dispatch <goal-id>` is the entry point: it checks Paths overlap against active/ongoing goals, creates the goal branch (from `origin/main` if it doesn't exist), and adds a sibling worktree. Refuses on overlap, missing Paths, or wrong status. See [`.claude/commands/goal-dispatch.md`](../.claude/commands/goal-dispatch.md).
