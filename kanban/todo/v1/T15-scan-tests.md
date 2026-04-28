# T15 · Scan Package — TDD Test Coverage Expansion

**Status:** todo  
**Version:** v1  
**Size:** large  
**Blocks:** T2 (validator uses scan), T4 (lint facade)

## What

Expand test coverage for `internal/scan` using TDD. A known golden-file test is currently
failing. Several exported utilities are untested and four utility stubs are unimplemented.
Work proceeds in four stages: analyse, sample fixtures, design, implement.

## Known Failure (fix first)

`TestScannerGoldenFiles/.entities_(largest_file)` — `scan_test.go:278`

The spot-check at offset 5765 is wrong — that byte is a TAB. The correct offset for
`NUMBER_LITERAL "20150324"` in that file is 25980. Fix this before adding anything new.

## Stage 1 — Analyse existing scan/ contents

Produce a coverage table for all exported symbols across the four source files:

| Symbol | File | Tested? | Notes |
|--------|------|---------|-------|
| `Scan` | scan.go | partial | happy path + unterminated quote/number; missing unknown-byte recovery |
| `IsWhitespace` | scan_util.go | no | |
| `IsDigit` | scan_util.go | no | |
| `IsNumberTypeSuffix` | scan_util.go | no | |
| `IsAlpha` | scan_util.go | no | |
| `IsIdentStart` | scan_util.go | no | |
| `IsIdentCont` | scan_util.go | no | |
| `IsASCIIPunct` | scan_util.go | yes | comprehensive boundary tests exist |
| `HumanSpan` | scan_util.go | no | |
| `Diagnostic.HumanString` | scan_util.go | no | |
| `FindTokensAtOffsets` | scan_util.go | yes | unit tested |
| `BuildLineIndex` | scan_util.go | yes | |
| `LineIndex.PosAt` | scan_util.go | yes | |
| `LineIndex.SpanPos` | scan_util.go | yes | |
| `Span.String` | scan_models.go | no | |
| `Diagnostic.String` | scan_models.go | no | |
| `Token.String` | scan_models.go | no | |
| `Kind.String` | scan_constants.go | no | |
| `Severity.String` | scan_constants.go | no | |
| `Lexeme` | scan_util.go | — | TODO stub, not implemented |
| `Sym` | scan_util.go | — | TODO stub, not implemented |
| `EqIdent` | scan_util.go | — | TODO stub, not implemented |
| `ParseIntLiteral` | scan_util.go | — | TODO stub, not implemented |

Update this table as stage 4 progresses.

## Stage 2 — Efficient fixture sampling

**Warning:** `void-files/` contains ~207K files, 3.5GB. Never read files >50KB without a
specific reason. Never glob the full tree.

Steps:
1. `find void-files/ -name '*.decl' -size -10k | head -30` — list small .decl candidates
2. `find void-files/ -name '*.decl' -size -10k | wc -l` — confirm volume
3. From the listing, choose 3–5 files spanning different directories (d2, doto,
   trimmed-Export) and size tiers
4. For each candidate, read it and confirm it is plain text and well-formed
5. Record chosen paths and their byte lengths for use in Stage 3

The 7.4MB `.entities` file currently used in golden tests is sufficient for the large-file
case. Do not add more multi-MB files.

## Stage 3 — Recommend improved test structure

Combine stages 1 and 2 to produce:

1. A proposed golden-file list: keep the 3 existing small .decl files; add 2–3 new small
   files from different directories to broaden character coverage
2. A revised spot-check table per golden file: offset, expected Kind, expected lexeme — all
   verified against actual file bytes before writing
3. A unit-test plan for uncovered Is* predicates (table-driven, boundary values)
4. A unit-test plan for String() display methods (one assertion per format string)
5. A plan for the 4 TODO stubs: write tests first (expected input→output), then implement

Document all proposed test names and their assertion strategy before writing a line of code.

## Stage 4 — Implementation (TDD order)

Execute in this order. Each sub-step: write the test first, confirm it fails, then implement.

**4a. Fix the failing test** (not TDD — fix the wrong datum)
- `scan_test.go` line ~180: change offset 5765 → 25980 for the "20150324" spot check
- Run `go test ./internal/scan/...` — suite must be green before continuing

**4b. Unit tests for uncovered Is* predicates**
- Add to `scan_models_test.go` (or a new `scan_util_test.go`)
- Table-driven tests with boundary values for `IsWhitespace`, `IsDigit`, `IsNumberTypeSuffix`,
  `IsAlpha`, `IsIdentStart`, `IsIdentCont`

**4c. Unit tests for String() display methods**
- `Span.String()` → `"[3,7)"`
- `Token.String()` → `"Ident@[3,7)"`
- `Diagnostic.String()` → check format contains code and span
- `Kind.String()` → all 6 kinds
- `Severity.String()` → all levels including DNE path

**4d. New golden-file tests**
- Add selected fixtures from Stage 2 to `TestScannerGoldenFiles`
- For each: call `validateTokenIntegrity()` and at least 3 spot-checked offsets (verified
  against actual bytes before writing)

**4e. Implement TODO stubs with tests first**
- `Lexeme(src []byte, tok Token) []byte` — slice of src for tok.Span
- `Sym(src []byte, tok Token) byte` — single byte at tok.Span.Start (panics if not SYMBOL)
- `EqIdent(src []byte, tok Token, lit string) bool` — bytes.Equal without alloc
- `ParseIntLiteral(src []byte, tok Token) (int64, error)` — parse NUMBER_LITERAL to int64

For each stub: write the test, confirm compile error or test failure, then implement.

## Dependencies

T0 (module scaffold, complete)  
T1 (parser, complete — uses scan; regressions must not appear)

## Verification

```
go test ./internal/scan/...   # all pass, zero failures
go test ./...                 # no regressions in parse or validate
```
