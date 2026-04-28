# T22 · Evaluate Subagents for Parallel Subtasks

**Status:** todo
**Version:** meta
**Size:** small

## What

Run a structured, collaborative trial of Claude Code parallel subagents on a real ticket with independent subtasks. Measure whether parallel execution materially reduces wall-clock time without degrading output quality or creating merge friction. Produce a documented adopt/reject decision agreed upon by Claude and the user together.

**This ticket must be run as a joint session with the user present** — not autonomously by Claude. The user observes the trial in real time and provides feedback throughout.

## Scope

### Pre-trial (define criteria before running anything)

Before starting the trial, Claude and the user agree on:

- **Speed threshold:** what speedup is worth the coordination overhead? (e.g., >25% wall-clock reduction)
- **Quality bar:** what counts as a subagent output failure? (e.g., broken tests, incorrect logic, missed scope items)
- **Merge complexity ceiling:** what level of merge effort is acceptable? (e.g., no conflicts vs. trivial conflicts only)
- **Sample size:** one ticket, or run on two separate tickets and compare?

### Trial design

- Identify the candidate ticket and its independent subtasks (recommended: T7, with a corpus-sweep pass and a binary-detection pass as separate subagents)
- Map each subtask to a dedicated subagent; assign each subagent its own worktree (T16 convention: `../void-slice-<branch>`)
- Define what each subagent receives as input and what it is expected to produce

### Execution

- Launch subagents in parallel within a single Claude Code session
- User observes and provides real-time feedback; Claude adapts if a subagent goes off-track
- Record actual wall-clock time for the parallel run vs. an estimated sequential baseline

### Post-trial debrief

- Compare actual outcomes against the pre-trial criteria
- Jointly assess: adopt, adopt with caveats, or reject
- Identify any caveats (e.g., "only for tasks with truly file-disjoint subtasks")

## Pre-trial Criteria (agreed)

- **Speed threshold:** any measurable wall-clock reduction (vs. estimated sequential baseline)
- **Quality bar:** all tests pass, no broken logic, no missed scope items
- **Merge complexity ceiling:** trivial conflicts acceptable (duplicate imports, blank lines); no structural conflicts
- **Sample size:** one ticket (T7)

**Trial candidate:** T7 — three file-disjoint test files as parallel subtasks:
- `internal/lint/clean_sweep_test.go` — clean file sweep (`.decl`, `.entitydef`, `.entities`, `.cfg`)
- `internal/lint/binary_sweep_test.go` — binary detection sweep (`.bwm`, `.tome`, etc.)
- `internal/lint/coverage_audit_test.go` — behavioral coverage audit (collect all diagnostic codes)

Each subagent runs in its own worktree (`../void-slice-<branch>`). T4 must be complete before the trial runs.

## Dependencies

T16 required — subagents writing files must each operate in an isolated worktree to avoid merge conflicts.  
T7 recommended as the trial candidate (corpus sweep + binary detection are file-disjoint).

## Verification

Run the trial; append a `## Completion` note with:
- Pre-trial criteria (filled in)
- Actual results
- Adopt/reject/caveats decision (agreed by user and Claude)

Document the decision in `CLAUDE.md § Tooling` (subagents entry) so future sessions know whether parallel subagents are an expected pattern in this repo.
