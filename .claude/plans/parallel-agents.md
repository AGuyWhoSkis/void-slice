# Parallel-agent PR workflow — planning scaffold

Goal: structure the repo so multiple Claude Code agents can run concurrently, each
owning one branch + one PR, with CI as the merge gate.

Pick this up cold — the four threads below are the menu. Each is independent enough
to ship on its own, but #1 unlocks the most leverage.

## Threads

### 1. Per-ticket branches + dispatch script

Today `/implement-ticket` stages on shared `m<N>-dev` branches, which re-serializes
parallel agents. Pivot to one branch per ticket, plus a `tools/dispatch-ticket.sh`
that creates the worktree, runs the agent, pushes, and opens the PR.

### 2. `Paths:` field + overlap check

Add a machine-readable `**Paths:**` field to ticket front-matter. Dispatcher rejects
spawning a ticket whose paths overlap an in-progress ticket. Cheap collision
prevention, useful even without #1.

### 3. Modular `Base:` field

Add `**Base:**` to tickets. Default `main`; override to `m<N>-dev` for goal-level
batching. Lets ticket → main and ticket → goal → main coexist per-ticket instead
of being a global mode.

### 4. Branch protection as code

Move the merge guardrail out of the GitHub UI into committed config — `CODEOWNERS`,
or a ruleset checked into `.github/`. The system-trust model only works if the
guardrail is durable.

## Out of scope

Sub-agents (Claude's in-process subagents) do not get their own branches. Existing
"self-contained, no git ops in parallel phase" rule already handles that boundary.

## Next session

Pick one or more threads, expand into tickets via `/goal-define` + `/goal-slice`.
