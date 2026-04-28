# T3 · Implement Report Rendering

**Status:** done  
**Version:** v1  
**Size:** medium  
**Blocks:** T5

## What

Implement the two output renderers in `internal/report/report.go`. No parsing or validation — pure formatting from `[]scan.Diagnostic` + source bytes.

## Scope

**Public API (expand the stub):**
```go
type RenderOptions struct {
    ContextLines int  // 1 recommended
    UseColor     bool // false in tests, true in CLI
}

func Render(src []byte, diags []scan.Diagnostic, opt RenderOptions) string
func RenderJSON(filename string, src []byte, diags []scan.Diagnostic) string
```

**Human-pretty format:**
```
file.decl:47:12 error [PARSE_UNTERMINATED_OBJECT] unterminated object block
  47 |   someDecl "foo" {
                        ^
```
- Use `LineIndex.SpanPos` (already in `scan_models.go`) to convert `Span` → `Pos`
- Clamp caret to line boundaries; zero-length span → single `^` at insertion point
- Sort diagnostics by `Span.Start` before rendering

**JSON format:**
```json
{
  "file": "file.decl",
  "diagnostics": [
    {"line": 47, "col": 12, "severity": "error", "code": "PARSE_UNTERMINATED_OBJECT", "message": "..."}
  ]
}
```
- Severity mapping: derive from code prefix (`VALIDATE_*` → `"warning"`, everything else → `"error"`) until T4 adds an explicit severity field
- Use `encoding/json` from stdlib; no external deps

**Tests (`internal/report/report_test.go`):**
- Snapshot/golden tests: for each broken fixture from T6, verify rendered human-pretty output matches expected file
- JSON tests: marshal → unmarshal round-trip; verify field values
- Edge cases: zero diagnostics, zero-length span, multi-line span clamped to one line

## Dependencies

T6 (broken testdata for snapshot tests); can otherwise be started immediately since it only needs `scan_models.go` types which are stable.

## Verification

```
go test ./internal/report/...
```

## Completion note

Implemented `Render` and `RenderJSON` in `internal/report/report.go`. Added `filename string`
as the first parameter to `Render` (ticket spec omitted it but the example output required it).
12 tests pass: 5 golden snapshot tests (one per T6 fixture), JSON round-trip, and 6 edge cases.
Golden files live in `internal/report/testdata/golden/`; regenerate with `-update` flag.
Pre-existing scan package failures (T15 scope) are unaffected.
