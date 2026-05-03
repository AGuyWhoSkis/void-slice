package lint_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"void-slice/internal/lint"
)

func lintSrc(t *testing.T, filename string, src []byte) []lint.Diagnostic {
	t.Helper()
	diags, err := lint.New().Lint(filename, src)
	require.NoError(t, err)
	return diags
}

func diagCodes(diags []lint.Diagnostic) []string {
	out := make([]string, len(diags))
	for i, d := range diags {
		out[i] = d.Code
	}
	return out
}

func TestCleanDecl(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntTest myComp {
		edit = {
			m_val = "hello";
		}
	}
}
`)
	diags := lintSrc(t, "test.decl", src)
	assert.Empty(t, diags)
}

func TestBinaryFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "binary", "sample.bwm")
	src, err := os.ReadFile(path)
	require.NoError(t, err)

	// Known binary extension → binary error before sniff
	diags := lintSrc(t, "sample.bwm", src)
	require.Len(t, diags, 1)
	assert.Equal(t, lint.Error, diags[0].Severity)
	assert.Equal(t, "LINT_BINARY_FILE", diags[0].Code)
}

func TestBinarySniff(t *testing.T) {
	// Unknown extension but contains null byte → binary error via sniff
	src := []byte("some text\x00more text")
	diags := lintSrc(t, "unknown.xyz", src)
	require.Len(t, diags, 1)
	assert.Equal(t, lint.Error, diags[0].Severity)
	assert.Equal(t, "LINT_BINARY_FILE", diags[0].Code)
}

func TestEntitiesWarning(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntTest myComp {
		edit = {
			m_val = "hello";
		}
	}
}
`)
	diags := lintSrc(t, "map.entities", src)
	require.NotEmpty(t, diags)
	assert.Equal(t, lint.Warning, diags[0].Severity)
	assert.Equal(t, "LINT_VE_INCONSISTENCY", diags[0].Code)
}

func TestDiagnosticCap_LintLevelCoversScan(t *testing.T) {
	// '@' is not a recognized byte in scan — each one emits one VOID_SCAN
	// "unknown byte" diagnostic. 20 of them with cap=5 proves the lint-level
	// cap covers scan output (which the parse / validate caps do not).
	src := bytes.Repeat([]byte{'@'}, 20)

	full, err := lint.New().Lint("input", src)
	require.NoError(t, err)
	scanCount := 0
	for _, d := range full {
		if d.Code == "VOID_SCAN" {
			scanCount++
		}
		assert.NotEqual(t, "PARSE_DIAGNOSTICS_TRUNCATED", d.Code,
			"no truncation sentinel when uncapped")
	}
	assert.Equal(t, 20, scanCount, "uncapped lint must surface every scan diagnostic")

	const cap = 5
	capped, err := lint.NewWithOptions(lint.Options{MaxDiagnostics: cap}).Lint("input", src)
	require.NoError(t, err)
	require.Len(t, capped, cap)
	assert.Equal(t, "PARSE_DIAGNOSTICS_TRUNCATED", capped[cap-1].Code,
		"final entry must be the truncation sentinel")
	assert.Equal(t, lint.Error, capped[cap-1].Severity)
	for i := 0; i < cap-1; i++ {
		assert.NotEqual(t, "PARSE_DIAGNOSTICS_TRUNCATED", capped[i].Code,
			"non-final entries must be original scan diagnostics")
	}
}

func TestBrokenDeclHasDiagnostics(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "broken", "count-mismatch.decl")
	src, err := os.ReadFile(path)
	require.NoError(t, err)

	diags := lintSrc(t, "count-mismatch.decl", src)
	codes := diagCodes(diags)
	assert.Contains(t, codes, "VALIDATE_ARRAY_COUNT_MISMATCH")
}
