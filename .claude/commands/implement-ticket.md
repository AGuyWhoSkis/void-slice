Implement ticket $ARGUMENTS end-to-end. Follow these steps in order:

1. **LOCATE** — find the ticket file in `kanban/{todo,in-progress}/`. Status folders are flat; the goal-id is encoded in the filename prefix (e.g. `M1.10-cli-polish.md`).

2. **READ** — read the full ticket. Understand the What, Scope, Dependencies, and Verification sections before doing anything else.

3. **DEPENDENCIES** — for each listed dependency, verify it is no longer present in `kanban/todo/` or `kanban/in-progress/` (closed tickets are deleted, so absence = done). If a dependency is still open in either folder, stop and report which ones.

4. **START** — edit the ticket's `**Status:**` field to `in-progress`. The kanban-move hook will move the file automatically.

5. **PLAN** — enter plan mode. Explore the codebase to understand what exists and what needs to change. Surface all ambiguous requirements as questions for the user. Write a concrete implementation plan. Do not write code until the user approves the plan.

6. **IMPLEMENT** — execute the approved plan. Follow project conventions in CLAUDE.md. Prefer editing existing files over creating new ones. Do not add features beyond the ticket scope.

7. **VERIFY** — run every verification step listed in the ticket. Confirm `go test ./...` passes and `go vet ./...` is clean. If integration tests (T7) exist, confirm they do not regress.

8. **GAPS** — before closing, review the work for gaps between the approved plan and what actually shipped. Propose follow-up tickets to the user; do not auto-create them. Check for, in priority order:

   - **Deferred work** — any TODO/FIXME/`.skip`/temporary workaround/feature flag added during this ticket
   - **Out-of-scope discoveries** — bugs, dead code, or broken invariants noticed but not fixed because they sat outside scope
   - **Plan deviations** — the approach diverged from the approved plan (different abstraction, skipped sub-task, unplanned helper). Each deviation is a candidate ticket if it leaves follow-up work
   - **Missing prereqs found late** — something the plan assumed existed but didn't; if a stopgap was added, file the proper fix as a ticket
   - **Verification gaps** — the ticket's Verification section didn't cover a real risk; propose a test/observability ticket
   - **New design pressure** — implementation revealed a refactor worth doing separately

   Present findings in a single batched prompt: "Found N follow-ups. Create tickets for: [one-line title + 2–3 sentence rationale per item]? (y / pick subset / n)". On approval, scaffold each via the PROPOSE format in `/crud-ticket` (each gets `**Origin:** <this-ticket-id>`). If no gaps, say so explicitly ("No follow-ups found.") so the user knows the step ran.

9. **CLOSE** — `git rm` the ticket file. Use the commit message as the closure record: summarise what was done, key decisions, and any deviations from the original scope. End the message with a `Follow-ups:` line listing the ticket IDs created in step 8 (or `none`).
