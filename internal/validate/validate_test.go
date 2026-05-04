package validate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"void-slice/internal/scan"
	"void-slice/internal/validate"
)

func scanAndValidate(t *testing.T, src []byte) []scan.Diagnostic {
	t.Helper()
	toks, scanDiags, _ := scan.Scan(src)
	require.Empty(t, scanDiags, "unexpected scan diagnostics")
	return validate.ValidateEntities("test.entities", src, toks)
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "broken", name)
	src, err := os.ReadFile(path)
	require.NoError(t, err)
	return src
}

func diagCodes(diags []scan.Diagnostic) []scan.DiagnosticCode {
	out := make([]scan.DiagnosticCode, len(diags))
	for i, d := range diags {
		out[i] = d.Code
	}
	return out
}

func hasCode(diags []scan.Diagnostic, code scan.DiagnosticCode) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// -------------------------
// Happy path
// -------------------------

func TestValidArray(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntTest myTest {
		edit = {
			m_items = {
				num = 2;
				item[0] = { m_val = "a"; }
				item[1] = { m_val = "b"; }
			}
		}
	}
}`)
	diags := scanAndValidate(t, src)
	assert.Empty(t, diags, "valid array should produce no diagnostics: %v", diags)
}

// -------------------------
// Fixture-based tests
// -------------------------

// Validator-layer golden coverage (M4.8): pin the full diagnostic shape —
// code, span, message — for each surviving validate warning pattern.
//
// Validate-package warnings after M4.8: ARRAY_INDEX_OOB, ARRAY_DUP_INDEX.
// COUNT_MISMATCH and MISSING_NUM were retired (G-B.1 follow-up for the
// inheritance-aware version).

func TestIndexOOB(t *testing.T) {
	src := loadFixture(t, "index-oob.decl")
	diags := scanAndValidate(t, src)
	require.Len(t, diags, 1, "expected exactly one diagnostic, got %v", diagCodes(diags))
	d := diags[0]
	assert.Equal(t, validate.Codes.ARRAY_INDEX_OOB, d.Code)
	assert.Equal(t, "array index 5 out of bounds [0, 2)", d.Message)
	// span anchors the offending indexer's value token (the literal `5`)
	assert.Equal(t, "5", string(src[d.Span.Start:d.Span.End]))
}

func TestDupIndex(t *testing.T) {
	src := loadFixture(t, "dup-index.decl")
	diags := scanAndValidate(t, src)
	require.Len(t, diags, 1, "expected exactly one diagnostic, got %v", diagCodes(diags))
	d := diags[0]
	assert.Equal(t, validate.Codes.ARRAY_DUP_INDEX, d.Code)
	assert.Equal(t, "duplicate array index 0", d.Message)
	// span anchors the duplicate indexer's value token (the second `0`)
	assert.Equal(t, "0", string(src[d.Span.Start:d.Span.End]))
}

func TestUnterminatedObject(t *testing.T) {
	src := loadFixture(t, "unterminated-object.decl")
	toks, _, _ := scan.Scan(src)
	diags := validate.ValidateEntities("test.entities", src, toks)
	assert.True(t, hasCode(diags, "PARSE_UNTERMINATED_OBJECT"),
		"expected PARSE_UNTERMINATED_OBJECT, got %v", diagCodes(diags))
}

func TestMissingSemicolon(t *testing.T) {
	src := loadFixture(t, "missing-semicolon.decl")
	diags := scanAndValidate(t, src)
	assert.True(t, hasCode(diags, "PARSE_EXPECTED_SEMICOLON"),
		"expected PARSE_EXPECTED_SEMICOLON, got %v", diagCodes(diags))
}

// -------------------------
// Inline negative tests
// -------------------------

// Per M4.8: COUNT_MISMATCH and MISSING_NUM retired. `num` is array capacity,
// not item count; sparse partial overrides of inherited arrays are legal in
// the corpus. Inheritance-aware re-introduction tracked in G-B.1.
func TestPartialOverrideAccepted(t *testing.T) {
	// num=4 with only item[0] populated is a legal partial override and must
	// not warn. Pre-M4.8 this would have fired ARRAY_COUNT_MISMATCH.
	src := []byte(`Version 1
component {
	cpntTest myTest {
		edit = {
			m_items = {
				num = 4;
				item[0] = { m_val = "a"; }
			}
		}
	}
}`)
	diags := scanAndValidate(t, src)
	assert.Empty(t, diags, "partial override should produce no diagnostics: %v", diags)
}

func TestSparseItemsWithoutNumAccepted(t *testing.T) {
	// item[...] entries with no num declaration is a legal full/partial
	// override (num inherited). Pre-M4.8 this would have fired
	// ARRAY_MISSING_NUM.
	src := []byte(`Version 1
component {
	cpntTest myTest {
		edit = {
			m_items = {
				item[0] = { m_val = "a"; }
			}
		}
	}
}`)
	diags := scanAndValidate(t, src)
	assert.Empty(t, diags, "sparse override should produce no diagnostics: %v", diags)
}

func TestNestedArraysIndependent(t *testing.T) {
	// Each nested array is validated independently.
	src := []byte(`Version 1
component {
	cpntTest myTest {
		edit = {
			m_outer = {
				num = 1;
				item[0] = {
					m_inner = {
						num = 2;
						item[0] = { m_x = 1; }
						item[1] = { m_x = 2; }
					}
				}
			}
		}
	}
}`)
	diags := scanAndValidate(t, src)
	assert.Empty(t, diags, "nested valid arrays should produce no diagnostics: %v", diags)
}
