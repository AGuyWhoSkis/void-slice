Implement ticket $ARGUMENTS end-to-end. Follow these steps in order:

1. **LOCATE** — find the ticket file in kanban/ (search todo/, in-progress/, and done/ subdirectories across all version folders).

2. **READ** — read the full ticket. Understand the What, Scope, Dependencies, and Verification sections before doing anything else.

3. **DEPENDENCIES** — for each listed dependency, verify it is in kanban/done/. If any dependency is not done, stop and report which ones are missing.

4. **START** — edit the ticket's `**Status:**` field to `in-progress`. The kanban-move hook will move the file automatically.

5. **PLAN** — enter plan mode. Explore the codebase to understand what exists and what needs to change. Surface all ambiguous requirements as questions for the user. Write a concrete implementation plan. Do not write code until the user approves the plan.

6. **IMPLEMENT** — execute the approved plan. Follow project conventions in CLAUDE.md. Prefer editing existing files over creating new ones. Do not add features beyond the ticket scope.

7. **VERIFY** — run every verification step listed in the ticket. Confirm `go test ./...` passes and `go vet ./...` is clean. If integration tests (T7) exist, confirm they do not regress.

8. **CLOSE** — edit the ticket's `**Status:**` field to `done`. Append a `## Completion` section: what was done, key decisions, any deviations from the original scope.

**Definition of done:** see CLAUDE.md § Definition of done.
