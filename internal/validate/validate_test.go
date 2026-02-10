package validate_test

// Planned: unit tests for array-shape checks (highest value).
//
// Fixtures (small snippets):
//
//   m_children = {
//     num = 2;
//     item[0] = { m_name = "a"; }
//     item[1] = { m_name = "b"; }
//   }
//
// Positive test:
//   - No validate diags.
//
// Negative tests:
//   [ ] num=2 but only item[0] present => count mismatch diag.
//   [ ] num=2 with item[2] present => index OOB diag.
//   [ ] duplicate item[0] twice => dup diag.
//   [ ] items present but missing num => warning (if implemented).
//
// Each test should:
//   - scan.Scan -> tokens (assert scan diags empty)
//   - validate.ValidateEntities -> diags (assert expected codes + spans)