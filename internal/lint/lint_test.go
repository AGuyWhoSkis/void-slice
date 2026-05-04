package lint_test

import (
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

// TestSidecarXMLIsNoOp — `.decl.xml` is recognized as Void Explorer export
// metadata. The lint layer skips the text scanner and emits nothing.
// Covers M4.7's drain of the residual sidecar PARSE+VOID_SCAN hits.
func TestSidecarXMLIsNoOp(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "golden",
		"generated.decls.animbasic.models.characters.dlc01.player.billie.additives_body..animbasic.decl.xml")
	src, err := os.ReadFile(path)
	require.NoError(t, err)

	diags := lintSrc(t, path, src)
	assert.Empty(t, diags, "expected zero diagnostics on .decl.xml sidecar; got %v", diags)
}

func TestBrokenDeclHasDiagnostics(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "broken", "count-mismatch.decl")
	src, err := os.ReadFile(path)
	require.NoError(t, err)

	diags := lintSrc(t, "count-mismatch.decl", src)
	codes := diagCodes(diags)
	assert.Contains(t, codes, "VALIDATE_ARRAY_COUNT_MISMATCH")
}
