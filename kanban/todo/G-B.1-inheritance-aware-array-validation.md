# G-B.1 · Inheritance-aware array validation

**Status:** todo
**Goal:** G-B (cross-file reference resolution; candidate goal per [M4.md](../goals/M4.md))
**Size:** medium

## What

Re-introduce array-shape validation that is actually useful to modders editing
real game files. Blocked on G-B (loading and resolving the `inherit = "..."`
chain) — the rule needs the resolved base array to do anything meaningful.

## Why this exists

M4.8 retired `VALIDATE_ARRAY_COUNT_MISMATCH` and `VALIDATE_ARRAY_MISSING_NUM`
because their premise was wrong for this format. They assumed `num = N` meant
"exactly N items in this block," but the corpus shows `num` is the array's
logical capacity and `item[i]` writes specific indices — sparse partial
overrides of an inherited base are normal and legal. All 147 corpus hits were
inside `edit = { ... }` blocks doing exactly this.

Net effect of M4.8: the rules' false-positive pattern (warns on legal
overrides) is gone. The genuinely useful detection — "modder forgot to update
`num` after adding/removing an item" — was never reachable without inheritance
context, because we can't tell "intentional partial override" from "forgot to
bump num" without knowing the base array's size.

## Scope (sketch — refine when G-B lands)

With the inherit chain resolvable, surface:
- `num` set to a value that disagrees with the resolved base array's size
- `item[i]` written at an index that is out of bounds in the resolved base
  (without an explicit `num` override raising the cap)
- Self-contained arrays (no inherit ancestor) where item count ≠ `num` —
  i.e. the original M4.8 rules, scoped to the case where they're actually
  meaningful

`VALIDATE_ARRAY_INDEX_OOB` and `VALIDATE_ARRAY_DUP_INDEX` survived M4.8 and
remain useful as-is; this ticket is purely additive.

## Dependencies

G-B (cross-file reference resolution). Until then, this ticket sits in the
parking lot; do not promote without G-B's file-graph layer in place.
