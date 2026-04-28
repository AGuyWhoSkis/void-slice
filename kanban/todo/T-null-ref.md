# T-null-ref · NULL; Reference Validation (Stretch)

**Status:** todo  
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
