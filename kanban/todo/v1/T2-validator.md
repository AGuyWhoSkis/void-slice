# T2 · Implement Validator

**Status:** todo  
**Version:** v1  
**Size:** large  
**Blocks:** T4

## What

Implement the frame-stack validator in `internal/validate/validate.go`. Two lint rules for v1. Requires T1 (parser) and T6 (broken testdata) to be meaningful to test.

## Scope

**Public API:**
```go
func ValidateEntities(src []byte, toks []scan.Token) (diags []scan.Diagnostic)
```
Internally creates a validator handler and calls `parse.WalkEntities`.

**Frame stack:**
- `OnObjectBegin` → push new `objFrame`
- `OnAssignment` → update top frame (detect `num`/`item[idx]` patterns)
- `OnObjectEnd` → run checks on frame, then pop

**Rule 1 — Array count/index validation (amber warning):**
- `num = X;` → record expected count and span on current frame
- `item[idx] = ...;` → record index in frame's items map; detect duplicates
- At `OnObjectEnd`: if `numVal` set, check `len(items) == numVal`, all indices in `[0, numVal-1]`, no duplicates
- Diagnostic codes: `VALIDATE_ARRAY_COUNT_MISMATCH`, `VALIDATE_ARRAY_INDEX_OOB`, `VALIDATE_ARRAY_DUP_INDEX`

**Rule 2 — Bracket/structural parity:**
- Unterminated objects are caught by the parser (T1) via `PARSE_UNTERMINATED_OBJECT`; the validator does not duplicate this
- If `item[...]` entries appear but no `num` key exists → optional `VALIDATE_ARRAY_MISSING_NUM` warning

**Severity (design note):**
- `scan.Diagnostic` has no severity field (it was removed in v0)
- Array warnings are amber; structural errors from parser are red
- Severity will live on `lint.Diagnostic` (T4), not on `scan.Diagnostic`
- For now, encode severity intent in the diagnostic code prefix: `VALIDATE_*` = warning unless the code name ends in `_ERROR`

**New diagnostic codes:**
- `VALIDATE_ARRAY_COUNT_MISMATCH`, `VALIDATE_ARRAY_INDEX_OOB`, `VALIDATE_ARRAY_DUP_INDEX`, `VALIDATE_ARRAY_MISSING_NUM`

**Explicitly out of scope for v1:**
- `VALIDATE_COMPONENT_MISSING_EDIT` (warn when a component block lacks an `edit` sub-block) — deferred; do not implement in T2. If the design doc in `validate.go` references it, remove or comment it out as "future rule."

**Tests (`internal/validate/validate_test.go`):**
- Table-driven against broken `.decl` fixtures from T6
- One test per rule: count mismatch, OOB index, duplicate index, missing-num warning
- Integration: run against clean golden files; expect zero validate diagnostics

## Dependencies

T1 (parser must be complete), T6 (broken testdata)

## Verification

```
go test ./internal/validate/...
go test ./internal/parse/...    # no regressions
```
