Slice goal $ARGUMENTS into the minimum set of tickets needed to ship it. `$ARGUMENTS` is a goal-id like `M8`.

Output: a set of ticket files in `kanban/todo/`.

---

## Rules

1. **The goal is immutable.** Read `kanban/goals/<goal-id>.md` for scope and intent, but do not edit it. If the goal seems wrong, surface that to the user — do not silently rewrite it.
2. **Each ticket is minimal and non-overlapping.** No two tickets should share scope, files, or deliverables. If a draft pair overlaps, merge them or redraw the seam before writing.
3. **Defer work to implementation time.** Each ticket's `## Scope` names *what must end up true*, not *how to get there*. Investigation, file paths, and design choices belong in the `/implement-ticket` PLAN step.

---

## Process

1. **Draft titles first.** Read the goal file. Propose the ticket list as titles + one-line summaries. Show it to the user. Iterate until the seams are clean and minimal.
2. **Write the files.** Once the list is approved, write each ticket under `kanban/todo/<goal-id>.<n>-<short-name>.md` using the CREATE shape from `/crud-ticket`. Numbering follows the rule there.
3. **Hand off.** Tell the user: "Ticket files written to `kanban/todo/`. Commit them on the per-goal integration branch (`m<N>-dev`, branched from `main` if it doesn't exist) before running `/implement-ticket` — that command requires a clean working tree." Do not commit on the user's behalf.
