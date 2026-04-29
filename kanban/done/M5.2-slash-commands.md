# T17 · Slash Commands for Recurring Prompt Patterns

**Status:** done  
**Version:** meta  
**Size:** small

## What

Encode the two most-repeated prompt scaffolds — "CRUD a ticket" and "plan/implement a ticket" — as slash commands in `.claude/commands/`. Eliminates retyping the same preamble at the start of every session.

## Scope

Create two files:

- `.claude/commands/crud-ticket.md` — prompt scaffold for creating, updating, or closing a ticket (frontmatter fields, folder placement, format rules)
- `.claude/commands/implement-ticket.md` — prompt scaffold for planning and implementing a ticket (read ticket → plan → implement → verify)

Each file should contain the full prompt text the slash command expands to, with `$ARGUMENTS` as the placeholder for variable input (e.g., ticket number or name).

Do NOT add a `/move-ticket` command — T23 automates ticket file movement via hook; the only manual step is editing the `**Status:**` field.

## Dependencies

- **T19 first** — slash commands that invoke shell ops need pre-approved commands; without the allowlist, every invocation prompts for permission, defeating the purpose
- **T23 alignment** — after T23 is implemented, `/crud-ticket` must only update the `**Status:**` field (one-line edit); the hook handles the `git mv` automatically. Update the command prompt at that point and remove any manual file-move instructions.
- **T20 alignment** — `/implement-ticket`'s "definition of done" step should reference the DoD section in CLAUDE.md rather than hardcoding it inline; draft that step as a placeholder until T20 exists

## Verification

```
/crud-ticket T17
/implement-ticket T4
```
Confirm each command loads the correct scaffold without manual copy-paste.

## Completion

Created `.claude/commands/crud-ticket.md` and `.claude/commands/implement-ticket.md`.

`/crud-ticket` covers create/update-status/add-completion-note/delete. Status updates instruct Claude to edit only the `**Status:**` field; the kanban-move hook handles `git mv` automatically.

`/implement-ticket` covers the full locate → read → deps-check → plan → implement → verify → close loop. DoD step references `CLAUDE.md § Definition of done` rather than hardcoding it inline. T23 alignment applied from the start.
