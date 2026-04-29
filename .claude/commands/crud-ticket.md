Read or manage the kanban ticket $ARGUMENTS. Search kanban/todo/, kanban/in-progress/, and kanban/done/ (including all version subfolders: meta, v1, v2, v3, stretch).

---

## CREATE a new ticket

Write a new file at kanban/todo/<version>/T<n>-short-name.md using this format:

```
# T<n> · <Title>

**Status:** todo
**Version:** <meta|v1|v2|v3|stretch>
**Size:** <small|medium|large>

## What
<one paragraph>

## Scope
<bullet list of specific, verifiable deliverables>

## Dependencies
<ticket numbers, or None>

## Verification
<how to confirm this ticket is complete>
```

After creating the file, confirm the ticket number does not collide with existing tickets.

---

## PROPOSE a follow-up ticket (gap discovered during another ticket)

Same as CREATE, with one addition: include an **Origin** field directly under **Size** so the gap-ticket traces back to the ticket that surfaced it.

```
# T<n> · <Title>

**Status:** todo
**Version:** <meta|v1|v2|v3|stretch>
**Size:** <small|medium|large>
**Origin:** T<originating-ticket>

## What
<one paragraph — include what was observed during T<origin> that motivated this ticket>

## Scope
<bullet list>

## Dependencies
<ticket numbers, or None>

## Verification
<how to confirm this ticket is complete>
```

Use this format when the trigger is `/implement-ticket`'s GAPS step or any other "noticed while doing X" situation.

---

## UPDATE STATUS

Edit **only** the `**Status:**` field — valid values: `todo`, `in-progress`, `done`.
The kanban-move hook automatically runs `git mv` to move the file to the correct folder.
**Do not manually move ticket files.**

---

## ADD A COMPLETION NOTE

Append a `## Completion` section at the bottom of the ticket. Include: what was done, key decisions, any deviations from original scope.

---

## DELETE a ticket

Confirm with the user before deleting. Prefer marking `done` over deletion.
