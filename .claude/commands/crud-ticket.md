Read or manage the kanban ticket $ARGUMENTS. Search kanban/todo/ and kanban/in-progress/ — both status folders are flat; goal-membership is encoded in the filename prefix (e.g. `M1.10-cli-polish.md`, `M5.7-subagents-eval.md`, `M3.4-lsp-followups.md`).

---

## CREATE a new ticket

Pick the goal-id from `kanban/goals/` (one of `M1`, `M2`, `M3`, `M4`, `M5`). Pick the next free `<n>` within that goal:

```
ls kanban/{todo,in-progress}/<goal-id>.* 2>/dev/null | wc -l
```

Add 1 to that count to get `<n>`. Then write a new file at `kanban/todo/<goal-id>.<n>-short-name.md` using this format:

```
# <goal-id>.<n> · <Title>

**Status:** todo
**Goal:** <M1|M2|M3|M4|M5>
**Size:** <small|medium|large>

## What
<one paragraph>

## Scope
<bullet list of specific, verifiable deliverables>

## Dependencies
<ticket IDs, or None>

## Verification
<how to confirm this ticket is complete>
```

After creating the file, confirm the ID does not collide with an existing ticket.

---

## PROPOSE a follow-up ticket (gap discovered during another ticket)

Same as CREATE, with one addition: include an **Origin** field directly under **Size** so the gap-ticket traces back to the ticket that surfaced it.

```
# <goal-id>.<n> · <Title>

**Status:** todo
**Goal:** <M1|M2|M3|M4|M5>
**Size:** <small|medium|large>
**Origin:** <originating-ticket-id>

## What
<one paragraph — include what was observed during <origin> that motivated this ticket>

## Scope
<bullet list>

## Dependencies
<ticket IDs, or None>

## Verification
<how to confirm this ticket is complete>
```

Use this format when the trigger is `/implement-ticket`'s GAPS step or any other "noticed while doing X" situation.

---

## UPDATE STATUS

Edit **only** the `**Status:**` field — valid values: `todo`, `in-progress`.
The kanban-move hook automatically runs `git mv` to move the file to the correct folder.
**Do not manually move ticket files.**

---

## CLOSE / DELETE a ticket

Closing a ticket = `git rm` it. The closure record is the commit message (and optional retro prose appended to the parent goal file). There is no `done` status or folder.
