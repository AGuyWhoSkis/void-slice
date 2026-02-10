package report_test

// Planned: snapshot-style tests for formatting.
//
//   - Provide src with a couple lines.
//   - Provide a diag span that points into the middle.
//   - Assert output string contains:
//       - correct line/col
//       - line text
//       - caret alignment
//
// Include edge cases:
//   [ ] diag span that crosses newline (clamp rendering)
//   [ ] zero-length span (missing token)
//   [ ] span at EOF
