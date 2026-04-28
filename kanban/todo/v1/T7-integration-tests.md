# T7 · Integration Tests Against void-files/ Corpus

**Status:** todo  
**Version:** v1  
**Size:** small  
**Blocks:** nothing (end-of-v1 quality gate)

## What

Run the completed lint engine against the real game files in `./void-files/` to prove the linter handles production data without false positives, panics, or crashes.

## Scope

Create `integration/integration_test.go` (or add a `TestCorpus` function in `internal/lint/lint_test.go`) that:

**Clean file sweep:**
- Walk all `*.decl`, `*.entitydef`, `*.entities`, `*.cfg` files under `./void-files/doto/game1/` and `./void-files/d2/game1/`
- Call `lint.New().Lint(filename, src)` on each
- Assert: no `Error`-severity diagnostics (warnings acceptable; false-positive errors are not)
- Assert: no panic (use `recover` in test helper if needed)

**Binary detection sweep:**
- Walk files with extensions `.bwm`, `.tome`, `.navmesh`, `.mapresources`, `.soundpropa`, `.bnavmesh`, `.bphysworld`, `.maprscreusechunk0` under `./void-files/`
- Assert: each returns exactly one diagnostic with code matching the binary/extension detection logic in T4
- Assert: no crash or hang on binary content

**File loading:**
- Load files at test time via `os.ReadFile`; do NOT embed or read inline
- If `./void-files/` is absent (e.g., CI without the corpus), skip with `t.Skip("void-files corpus not present")`

**Behavioral coverage audit:**
- After both sweeps pass, collect every distinct diagnostic code emitted across the corpus (aggregate from all returned `[]lint.Diagnostic`)
- Cross-reference the collected codes against documented rules: T2 (array count/index), T4 (binary detection, VE-inconsistency warning), and any stretch rules
- Log the full code inventory (e.g. via `t.Logf`) so it is visible in verbose test output
- Flag any code with no corresponding ticket as a gap — open a follow-up ticket in `kanban/todo/v1/` or `kanban/todo/v2/` before closing V1

## Dependencies

T4 (lint facade must be complete — this test imports only `internal/lint`)

## Verification

```
go test ./integration/...     # or go test ./internal/lint/... -run TestCorpus
# expected: PASS, no errors, any warnings noted but not failing
```

Run manually after T4 completes. A single false-positive error on a known-good file is a blocker for T5.
