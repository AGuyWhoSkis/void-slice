# T6 · Testdata — Broken Fixture Files

**Status:** todo  
**Version:** v1  
**Size:** small  
**Blocks:** T2 (tests), T3 (snapshot tests)

## What

Create 4–6 hand-crafted broken `.decl` fixture files so T2 and T3 have real inputs to test against. All current golden files produce zero diagnostics — broken files are needed to drive test-driven development.

## Scope

Create a `testdata/broken/` directory (or `void-files/broken/`) with at least:

| File | Triggers |
|------|----------|
| `count-mismatch.decl` | `num = 3` but only 2 `item[n]` entries → `VALIDATE_ARRAY_COUNT_MISMATCH` |
| `index-oob.decl` | `num = 2`, `item[5]` exists → `VALIDATE_ARRAY_INDEX_OOB` |
| `dup-index.decl` | Two `item[0]` assignments in same object → `VALIDATE_ARRAY_DUP_INDEX` |
| `unterminated-object.decl` | Opening `{` with no closing `}` → `PARSE_UNTERMINATED_OBJECT` |
| `missing-semicolon.decl` | Assignment without trailing `;` → `PARSE_EXPECTED_SEMICOLON` |

Base each file on the structure of a real `.decl` golden file (copy a small clean one and introduce the single defect).

For each broken file, create a matching `<name>.expected.txt` and `<name>.expected.json` containing the expected `report.Render` and `report.RenderJSON` output. These are the snapshot files T3 tests compare against.

**Existing binary fixtures:** `void-files/binwalk test/` already exists with `.bwm`, `.tome`, `.entities` binaries — reuse as-is for T4 binary detection tests.

## Dependencies

None — can be done in parallel with T1.

## Verification

- Open each broken file and confirm the defect is clearly visible
- A human reading the file can predict exactly which diagnostic it should produce
