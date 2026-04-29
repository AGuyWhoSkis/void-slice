# T-null-ref · NULL; Reference Validation (Stretch)

**Status:** done
**Version:** v1 stretch (evaluate at end-of-week-1 velocity check)  
**Size:** small

## What

A third lint rule: flag `NULL;` used as a value in a context where a concrete reference is expected. README-v1.md identifies this as a potentially near-free rule given the parsing work already done.

## Scope

- During `OnAssignment`, check if the value token is the identifier `NULL`
- Emit `VALIDATE_NULL_REFERENCE` warning on the assignment span
- The rule is intentionally conservative: only flag `key = NULL;` at the top level of an object, not nested uses where `NULL` may be a valid sentinel

**Prerequisite check before starting:**
- Scan the void-files/ corpus (T7) for `NULL;` occurrences and assess whether flagging all of them would produce false positives
- If false-positive rate looks significant, backlog this rule to v2

**New diagnostic code:** `VALIDATE_NULL_REFERENCE`

## Dependencies

T2 (validator frame stack — add this rule to the validator), T7 (corpus test to verify no false positives)

## Verification

```
# broken fixture: null-ref.decl with `someKey = NULL;`
go test ./internal/validate/... -run TestNullRef
```

Gate: zero false positives on the void-files/ corpus sweep.

## Completion

Closed on 2026-04-28 — absorbed into the v4 refinement conversation. Status:

- The rule itself (a 14th lint code, `VALIDATE_NULL_REFERENCE`) may still be worth landing.
- The original gate is dead: the `void-files/` corpus referenced by the prerequisite check was retired in [T29-void-files-reorg](../meta/T29-void-files-reorg.md). The replacement corpus is `testdata/corpus-mini/`, but rewriting the gate against it without first deciding whether the rule is wanted is wasted work.
- Triage is therefore parked as **open question #3** in [T35-v4-linter-scope](../../todo/v4/T35-v4-linter-scope.md), where the user will decide between (a) absorb into v4 with a corpus-mini gate, (b) absorb into v4 without a gate, or (c) won't-do, during the v4 refinement conversation.

This ticket closes here so the backlog stops carrying a stretch item with a stale gate. The decision lives in T35.
