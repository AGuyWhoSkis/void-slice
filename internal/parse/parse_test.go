package parse_test

// Planned: tests that prove correctness on small snippets (unit-style).
//
// Test strategy:
//   - Provide a small .entities snippet as []byte with:
//       Version 6
//       component { cpntX name { edit = { m_a = 1; } } }
//   - Run scan.Scan -> tokens, scanDiags (assert no scanDiags for happy path)
//   - Run parse.WalkEntities with a test handler that records events.
//   - Assert event sequence:
//
//     OnVersion(6)
//     OnComponentBegin
//     OnComponentDecl(typeTok="cpntX", nameTok="name")
//     OnObjectBegin/End counts match expected nesting
//     OnAssignment keys/values captured correctly
//
// Add negative tests:
//   [ ] Missing ';' after assignment => parse diag emitted, recovery continues.
//   [ ] Missing '}' => parse diag emitted.
//   [ ] item[0] parsing: key base "item", index int 0.
//   [ ] item_add["abc"] parsing: key base "item_add", index string "abc".
//
// Performance sanity (not microbench):
//   - Optional: parse a medium fixture and ensure it completes quickly.