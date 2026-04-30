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

## Goal completion — context drain

When a goal completes, its done tickets are compacted into a short retrospective on the goal file, then deleted. Git keeps the implementation truth (commits + diffs); the goal file keeps the planning truth — why we did the work, what shipped, what was non-obvious.

The trigger is goal completion, never time- or count-based. That's deliberate: tickets only drain when a goal closes, so letting half-implemented goals linger has a real cost — bloat doesn't go away until the goal does.

### Retrospective format (v1)

Append a `## Retrospective` section to the goal file (`kanban/goals/M{N}.md`):

```markdown
## Retrospective

**What shipped.** 1–3 bullets covering the body of work — not per-ticket.

**What surprised us.** Non-obvious findings that came up during execution.

**Worth remembering.** Patterns, decisions, or pitfalls to carry into the next goal.
```

Out of v1: per-ticket recaps (`git log` has them), dates, owners, effort estimates. Marked v1 because it will evolve from real use — adjust the format as the first few real retros teach us what's missing or excess.

### Compaction checklist

When the last ticket of a goal lands in `done/`:

1. Write the `## Retrospective` section on `kanban/goals/M{N}.md` using the v1 format above.
2. Remove the `## Tickets` table from the goal file — the retrospective replaces it, and the per-ticket links would otherwise dangle once the files are gone.
3. `git rm kanban/done/M{N}.*` — every ticket file for that goal.
4. Flip the goal file's `**Status:**` field to `done`.
5. Commit. The single diff (retro added, table dropped, ticket files removed, status flipped) is the durable record of compaction.

No hook, slash command, or other tooling — just the manual checklist. If running it becomes routine enough that the friction matters, that's the signal to promote it.
