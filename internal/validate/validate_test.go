package validate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"void-slice/internal/parse"
	"void-slice/internal/scan"
	"void-slice/internal/validate"
)

func scanAndValidate(t *testing.T, src []byte) []scan.Diagnostic {
	t.Helper()
	toks, scanDiags, _ := scan.Scan(src)
	require.Empty(t, scanDiags, "unexpected scan diagnostics")
	return validate.ValidateEntities(src, toks, parse.Opts{})
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
// Diagnostic-count cap (M7.1)
// -------------------------

func TestDiagnosticCap_ValidateTruncated(t *testing.T) {
	// Input parses cleanly but produces many ARRAY_DUP_INDEX warnings — one
	// per duplicate `item[0]` line beyond the first.
	const dups = 20
	var b strings.Builder
	b.WriteString("Version 1\ncomponent {\n  cpntFoo myFoo {\n    arr = {\n      num = 1;\n")
	for i := 0; i < dups; i++ {
		fmt.Fprintf(&b, "      item[0] = { m_x = %d; }\n", i)
	}
	b.WriteString("    }\n  }\n}\n")
	src := []byte(b.String())

	toks, scanDiags, _ := scan.Scan(src)
	require.Empty(t, scanDiags)

	// Default Opts: every dup diagnostic emitted, no truncation.
	full := validate.ValidateEntities(src, toks, parse.Opts{})
	dupSeen := 0
	for _, d := range full {
		if d.Code == validate.Codes.ARRAY_DUP_INDEX {
			dupSeen++
		}
		assert.NotEqual(t, parse.Codes.DIAGNOSTICS_TRUNCATED, d.Code,
			"no truncation sentinel when uncapped")
	}
	assert.Equal(t, dups-1, dupSeen, "expected one dup per repeat after the first")

	// Capped: combined output is exactly cap; final entry is the sentinel.
	const cap = 5
	capped := validate.ValidateEntities(src, toks, parse.Opts{MaxDiagnostics: cap})
	require.Len(t, capped, cap)
	assert.Equal(t, parse.Codes.DIAGNOSTICS_TRUNCATED, capped[cap-1].Code,
		"final entry must be the truncation sentinel")
	for i := 0; i < cap-1; i++ {
		assert.NotEqual(t, parse.Codes.DIAGNOSTICS_TRUNCATED, capped[i].Code,
			"non-final entries must be original validate diagnostics")
	}
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
	diags := validate.ValidateEntities(src, toks, parse.Opts{})
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
