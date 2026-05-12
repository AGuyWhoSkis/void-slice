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

// TestShaderDeclIsNoOp — `.decl` files whose body is an inline raw-HLSL
// `hlsl_prefix { ... }` block are auto-generated wrappers around shader
// source. The HLSL preprocessor bytes (`#include <…>`, `#define`, `#if`)
// live entirely in inter-token gaps and produce one VOID_SCAN per byte —
// 148 / 116 hits on these two files. M12.18 routes them away from the
// scanner via a content sniff; lint returns empty diags.
func TestShaderDeclIsNoOp(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "testdata", "golden",
			"generated.decls.renderprog.tlf.gatherdepthminmax.decl"),
		filepath.Join("..", "..", "testdata", "golden",
			"generated.decls.renderprog.arksssblur.decl"),
	}
	for _, p := range paths {
		src, err := os.ReadFile(p)
		require.NoError(t, err)
		diags := lintSrc(t, p, src)
		assert.Empty(t, diags, "expected zero diagnostics on shader-prefix .decl %s; got %v", p, diags)
	}
}

func TestBrokenDeclHasDiagnostics(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "broken", "index-oob.decl")
	src, err := os.ReadFile(path)
	require.NoError(t, err)

	diags := lintSrc(t, "index-oob.decl", src)
	codes := diagCodes(diags)
	assert.Contains(t, codes, "VALIDATE_ARRAY_INDEX_OOB")
}
