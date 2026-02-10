package validate

// Planned: semantic validation that requires structure. This should be a thin layer that
// hooks into parse.WalkEntities via a Handler implementation.
//
// Recommended public API:
//
//   func ValidateEntities(src []byte, toks []scan.Token) (diags []scan.Diagnostic) {
//     // creates validator handler
//     // calls parse.WalkEntities(src, toks, handler)
//     // returns combined diags (parse + validate)
//   }
//
// Core idea: maintain an object-frame stack while walking nested { ... }.
//
//   type objFrame struct {
//     // Array-shape tracking:
//     numTok   *scan.Token
//     numVal   *int64
//     items    map[int64]scan.Token    // item index -> token span anchor
//     itemDup  map[int64]scan.Token    // for duplicate detection (optional)
//     // Track presence of keys if “required keys” checks are added later:
//     seenKeys map[string]scan.Token
//   }
//
//   type validator struct {
//     src []byte
//     frames []objFrame
//     diags []scan.Diagnostic
//   }
//
// Hook points:
//
//   OnObjectBegin: push new frame
//   OnAssignment: update top frame (detect num/item)
//   OnObjectEnd: run checks on that frame, then pop
//
// Array-shape detection:
//
//   When OnAssignment(key, value) fires:
//     - base := lexeme(key.BaseTok)
//     - if base == "num" and value.Kind==ValNumber:
//         parse int -> frame.numVal
//         frame.numTok = &key.BaseTok (or value.Tok for better pointing)
//     - if base == "item" and key has exactly one Indexer of Kind IndexInt:
//         idx := indexer.IntValue
//         record idx in frame.items (detect duplicates)
//
// At OnObjectEnd:
//   if frame.numVal != nil:
//     - expected := *frame.numVal
//     - actual := len(frame.items)
//     - checks:
//         [ ] actual == expected  (if not: diag at numTok/valueTok)
//         [ ] all indices in [0, expected-1]  (diag at offending item token)
//         [ ] no duplicates (diag at duplicate item token)
//         [ ] optionally: contiguous coverage (0..expected-1) (diag at numTok + list missing)
//   else:
//     - optionally: if item[...] entries exist but no num => warn
//
// Component-level validations (add after arrays work):
//   - Track when inside a component decl; ensure "edit = { ... }" exists somewhere.
//   - Track Version value acceptable range if needed.
//
// Diagnostic codes (validate layer):
//   - VALIDATE_ARRAY_COUNT_MISMATCH
//   - VALIDATE_ARRAY_INDEX_OOB
//   - VALIDATE_ARRAY_DUP_INDEX
//   - VALIDATE_ARRAY_MISSING_NUM (optional warning)
//   - VALIDATE_COMPONENT_MISSING_EDIT (optional warning)
//
// Spans:
//   - Count mismatch: span of num value token if available; else num key token.
//   - OOB/dup: span of the item[...] key token (or the indexer token).
//
// Milestones:
//   1) Wire validate.ValidateEntities to parse.WalkEntities (no rules yet).
//   2) Implement frame stack with OnObjectBegin/End.
//   3) Add num/item rules + tests.
//   4) Add “missing edit” rule if it’s reliably required in the corpus.
