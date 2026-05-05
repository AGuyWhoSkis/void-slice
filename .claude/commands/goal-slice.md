Slice goal $ARGUMENTS into the minimum set of tickets needed to ship it. `$ARGUMENTS` is a goal-id like `M8`.

Output: a set of contract-grade ticket files in `kanban/todo/` — tickets sharp enough that an agent can pick one up and ship a green PR without re-litigating design.

---

## Rules

1. **The goal is immutable.** Read `kanban/goals/<goal-id>.md` for scope and intent, but do not edit it. If the goal seems wrong, surface that to the user — do not silently rewrite it.
2. **Each ticket is minimal and non-overlapping.** No two tickets should share scope, files, or deliverables. If a draft pair overlaps, merge them or redraw the seam before writing. The seam check is mechanical: name the file paths each ticket will edit and verify the sets are disjoint (or that any shared file is split into clean before/after edits with one ticket strictly preceding the other).
3. **Bake in human decisions before dispatch.** Each ticket is a contract. File paths, format choices, edge-case handling, and library choices that affect the contract are decided *here*, not deferred to implementation. Deferrals are explicit and reasoned, in the ticket's `## Out of scope (intentionally deferred)` section — not implicit.

---

## Process

1. **Draft titles first.** Read the goal file. Propose the ticket list as titles + one-line summaries. Show it to the user. Iterate until the seams are clean and minimal.

2. **Per-ticket disambiguation pass.** For each draft ticket, before writing the file:
   - **Enumerate open decisions.** What's underspecified? What format / file path / library / edge case could reasonably go two ways?
   - **Recommend with reasoning by default.** Pick the better option and explain why. Trust the recommendation unless the user redirects.
   - **Ask only when input genuinely matters.** Don't ask "where should this live?" when one location is obviously right; don't surface every micro-choice. Only forks that affect downstream work — different tickets, different verification, different blast radius — warrant a question. Hedging questions erode the user's time.
   - **Catch seam violations.** If two tickets would edit the same file in incompatible ways (one expects shape A, the other shape B), redraw the seam now. Surfacing this at slice time is cheap; surfacing it during parallel dispatch is expensive.

3. **Write the files.** Once decisions are baked, write each ticket under `kanban/todo/<goal-id>.<n>-<short-name>.md` using the CREATE shape from `/crud-ticket`. Numbering follows the rule there.

4. **Adopt nits.** Run after the slicer's intentional tickets are written so the goal's own seams are settled first.

   1. `git fetch origin nits 2>/dev/null`. If the branch doesn't exist, print `(no nits substrate)` and skip the rest of this step.
   2. List pending nits in the same table format as `/goal-define`'s **Pending nits** step, marking each with ✓/✗ for "fits this goal" — i.e. every glob in the nit's `Paths:` is a subset of the adopting goal's `Paths:`.
   3. Prompt: `Adopt fitting nits? (all / none / pick subset)`. Non-fitting nits are listed for context but not offered.
   4. For each adopted nit, call `tools/nit-claim.sh <goal-id> <nit-filename>` (filename is the basename, no path).
      - Exit 0 (win): write a ticket at `kanban/todo/<goal-id>.<n>-<slug>.md` using the materialized ticket format below. `<n>` is the next free number per `/crud-ticket`'s rule, computed *after* the slicer's own tickets are on disk so adopted nits get numbers past them. `<slug>` is the same slug used in the nit's filename.
      - Exit 2 (race lost): print `<file> already claimed by another goal; skipping` and continue with the next nit.
      - Exit 3 (network failure): surface to the user and stop the adoption step. Already-won claims stay won; remaining adoptions can be retried by re-running `/goal-slice`.

   **Materialized ticket format** (literal, copyable):

   ```markdown
   # <goal-id>.<n> · <description from nit>

   **Status:** todo
   **Goal:** <goal-id>
   **Size:** small
   **Origin:** nit-<timestamp>

   ## What
   <captured prose from nit body — context paragraph if present, else just restate the description. Always include `Originating branch: <branch>` line.>

   ## Scope
   - <captured Paths from nit file, repeated as concrete deliverables>

   ## Dependencies
   None.

   ## Verification
   <fill this in during the disambiguation pass; if no clear verification can be named, the nit's adoption is reverted (`git -C ../void-slice-nits revert <claim-commit>`) rather than shipping a ticket without a check>

   ## Out of scope (intentionally deferred)
   None.
   ```

5. **Hand off.** Tell the user: "Ticket files written to `kanban/todo/`. Commit them on the goal branch (`goal/M<N>-<slug>`, created via `/goal-dispatch` if it doesn't exist) before running `/implement-ticket` — that command requires a clean working tree." Do not commit on the user's behalf.

---

## What "contract-grade" looks like

- **Concrete file paths in `## Scope`.** Not "the dispatcher script" — `tools/goal-dispatch.sh`. Not "the slash-command file" — `.claude/commands/goal-dispatch.md`.
- **Format examples in code blocks.** If the ticket adds a new field, show the field's literal markdown / JSON. If it adds a CLI flag, show the invocation. The implementer copies, doesn't infer.
- **Mechanically checkable verification.** "`grep -rn 'foo'` returns zero hits" is checkable. "It feels clean" is not. Each verification bullet should map to a command or a file inspection a third party could run.
- **Out-of-scope section names what's deferred and why.** "Auto-rebasing — surface only, never silent rewrite" is reasoned deferral. A blank `## Out of scope` invites scope creep.

The M9.1–M9.6 ticket files (or their `git log` ancestors if closed) are the reference exemplars.
