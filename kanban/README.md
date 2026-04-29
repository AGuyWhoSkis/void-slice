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

## Columns

| Folder | Meaning |
|--------|---------|
| `todo/` | Prioritized and ready to start |
| `in-progress/` | Being worked on right now |
| `done/` | Finished — has a `## Completion` note at the bottom |

Status folders are flat. Goal-membership is encoded in the filename prefix.

## Ticket format

Filename: `<goal-id>.<n>-short-name.md`. Examples: `M1.10-cli-polish.md`, `M5.7-subagents-eval.md`, `M3.4-lsp-followups.md`.

Edit the `**Status:**` field (`todo` / `in-progress` / `done`) — the kanban-move hook moves the file into the matching folder. Do not rename or `git mv` manually.

New tickets pick the next free `<n>` within their goal:

```
ls kanban/{todo,in-progress,done}/<goal-id>.* 2>/dev/null | wc -l
```

Add 1 to that count.
