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

func TestCountMismatch(t *testing.T) {
	src := loadFixture(t, "count-mismatch.decl")
	diags := scanAndValidate(t, src)
	assert.True(t, hasCode(diags, validate.Codes.ARRAY_COUNT_MISMATCH),
		"expected VALIDATE_ARRAY_COUNT_MISMATCH, got %v", diagCodes(diags))
}

func TestIndexOOB(t *testing.T) {
	src := loadFixture(t, "index-oob.decl")
	diags := scanAndValidate(t, src)
	assert.True(t, hasCode(diags, validate.Codes.ARRAY_INDEX_OOB),
		"expected VALIDATE_ARRAY_INDEX_OOB, got %v", diagCodes(diags))
}

func TestDupIndex(t *testing.T) {
	src := loadFixture(t, "dup-index.decl")
	diags := scanAndValidate(t, src)
	assert.True(t, hasCode(diags, validate.Codes.ARRAY_DUP_INDEX),
		"expected VALIDATE_ARRAY_DUP_INDEX, got %v", diagCodes(diags))
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

func TestMissingNum(t *testing.T) {
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
	assert.True(t, hasCode(diags, validate.Codes.ARRAY_MISSING_NUM),
		"expected VALIDATE_ARRAY_MISSING_NUM, got %v", diagCodes(diags))
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
