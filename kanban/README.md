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
