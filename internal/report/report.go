package report

// Planned: convert []Diagnostic into human-readable strings with line/col + context snippet.
// This should be independent of parse/validate; it only needs src + diags.
//
// Public API suggestion:
//
//   type RenderOptions struct {
//     ContextLines int // e.g. 1 or 2
//     UseColor bool   // optional; off by default for tests
//   }
//
//   func Render(src []byte, diags []scan.Diagnostic, opt RenderOptions) string
//   func RenderOne(src []byte, diag scan.Diagnostic, opt RenderOptions) string
//
// Implementation approach:
//   - Build scan.LineIndex once (already exists / planned).
//   - Convert diag.Span into Pos{Line,Col} via LineIndex.SpanPos.
//   - Render format example:
//
//       ERROR [SCAN/...] line 12, col 8: unterminated quote
//         12 |   m_name = "Root;
//                    ^^^^^^^^^
//
// For caret ranges:
//   - Use Span.Start..Span.End; clamp to line boundaries.
//   - If span is zero-length (missing expected token), render caret at insertion point.
//
// Keep this package pure formatting:
//   - No parsing.
//   - No validation.
//   - No file IO.
//
// Milestones:
//   1) RenderOne with line/col only.
//   2) Add single-line snippet + caret.
//   3) Add N context lines.
//   4) Add stable ordering (by span start then severity).
