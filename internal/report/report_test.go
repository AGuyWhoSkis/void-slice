package report_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"void-slice/internal/report"
	"void-slice/internal/scan"
	"void-slice/internal/validate"
)

var updateGolden = flag.Bool("update", false, "overwrite golden files with current output")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

// lintFixture runs scan + validate and returns all diagnostics.
func lintFixture(t *testing.T, path string, src []byte) []scan.Diagnostic {
	t.Helper()
	toks, scanDiags, _ := scan.Scan(src)
	validateDiags := validate.ValidateEntities(path, src, toks)
	return append(scanDiags, validateDiags...)
}

// loadFixture reads a broken fixture from testdata/broken/ relative to the project root.
// Tests run with cwd = package dir (internal/report/), so go up two levels.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "broken", name+".decl")
	src, err := os.ReadFile(path)
	require.NoError(t, err, "reading fixture %s", name)
	return src
}

func goldenPath(name string) string {
	return filepath.Join("testdata", "golden", name+".txt")
}

func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := goldenPath(name)
	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(got), 0644))
		return
	}
	want, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Fatalf("golden file missing: %s — run with -update to create it", path)
	}
	require.NoError(t, err)
	assert.Equal(t, string(want), got)
}

// --- Golden snapshot tests ---

func goldenTest(t *testing.T, fixtureName string) {
	t.Helper()
	src := loadFixture(t, fixtureName)
	diags := lintFixture(t, fixtureName+".decl", src)
	got := report.Render(fixtureName+".decl", src, diags, report.RenderOptions{})
	checkGolden(t, fixtureName, got)
}

func TestGolden_DupIndex(t *testing.T)           { goldenTest(t, "dup-index") }
func TestGolden_IndexOOB(t *testing.T)           { goldenTest(t, "index-oob") }
func TestGolden_MissingSemicolon(t *testing.T)   { goldenTest(t, "missing-semicolon") }
func TestGolden_UnterminatedObject(t *testing.T) { goldenTest(t, "unterminated-object") }

// --- JSON round-trip test ---

func TestRenderJSON_RoundTrip(t *testing.T) {
	src := []byte("num = 3;\nitem[0] { }\n")
	diags := []scan.Diagnostic{
		{Code: "VALIDATE_ARRAY_INDEX_OOB", Span: scan.NewSpan(6, 7), Message: "out of bounds"},
		{Code: "VOID_SCAN", Span: scan.NewSpan(9, 13), Message: "bad token"},
	}
	got := report.RenderJSON("x.decl", src, diags)

	var out struct {
		File        string `json:"file"`
		Diagnostics []struct {
			Line     int    `json:"line"`
			Col      int    `json:"col"`
			Severity string `json:"severity"`
			Code     string `json:"code"`
			Message  string `json:"message"`
		} `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal([]byte(got), &out))

	assert.Equal(t, "x.decl", out.File)
	require.Len(t, out.Diagnostics, 2)

	// sorted by Span.Start: VALIDATE diag first (offset 6), VOID_SCAN second (offset 9)
	assert.Equal(t, "warning", out.Diagnostics[0].Severity)
	assert.Equal(t, "VALIDATE_ARRAY_INDEX_OOB", out.Diagnostics[0].Code)
	assert.Equal(t, 1, out.Diagnostics[0].Line)
	assert.Equal(t, 7, out.Diagnostics[0].Col)

	assert.Equal(t, "error", out.Diagnostics[1].Severity)
	assert.Equal(t, "VOID_SCAN", out.Diagnostics[1].Code)
	assert.Equal(t, 2, out.Diagnostics[1].Line)
}

// --- Edge case unit tests ---

func TestRender_ZeroDiagnostics(t *testing.T) {
	got := report.Render("x.decl", []byte("hello"), nil, report.RenderOptions{})
	assert.Equal(t, "", got)
}

func TestRender_ZeroLengthSpan(t *testing.T) {
	src := []byte("hello world\n")
	diags := []scan.Diagnostic{{Code: "VOID_SCAN", Span: scan.NewSpan(5, 5), Message: "oops"}}
	got := report.Render("x.decl", src, diags, report.RenderOptions{})

	lines := strings.Split(got, "\n")
	caretLine := lines[len(lines)-1]
	assert.Equal(t, 1, strings.Count(caretLine, "^"), "zero-length span should produce exactly one caret")
}

func TestRender_MultilineSpanClamped(t *testing.T) {
	src := []byte("line one\nline two\n")
	// Span [0, 17) crosses both lines; caret must not exceed "line one" length (8)
	diags := []scan.Diagnostic{{Code: "VOID_SCAN", Span: scan.NewSpan(0, 17), Message: "big span"}}
	got := report.Render("x.decl", src, diags, report.RenderOptions{})

	lines := strings.Split(got, "\n")
	caretLine := lines[len(lines)-1]
	assert.LessOrEqual(t, strings.Count(caretLine, "^"), len("line one"))
}

func TestRenderJSON_EmptyDiagnostics(t *testing.T) {
	got := report.RenderJSON("x.decl", []byte("hello"), nil)

	var out struct {
		File        string          `json:"file"`
		Diagnostics []interface{}   `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal([]byte(got), &out))
	assert.Equal(t, "x.decl", out.File)
	assert.NotNil(t, out.Diagnostics, "diagnostics should be [] not null")
	assert.Empty(t, out.Diagnostics)
}

func TestRender_SeverityMapping(t *testing.T) {
	src := []byte("x = 1;\ny = 2;\n")
	diags := []scan.Diagnostic{
		{Code: "VALIDATE_ARRAY_INDEX_OOB", Span: scan.NewSpan(0, 1), Message: "out of bounds"},
		{Code: "VOID_SCAN", Span: scan.NewSpan(7, 8), Message: "bad byte"},
	}
	got := report.Render("x.decl", src, diags, report.RenderOptions{})
	assert.Contains(t, got, "warning")
	assert.Contains(t, got, "error")
}

func TestRender_SortOrder(t *testing.T) {
	src := []byte("aaa bbb ccc\n")
	// Supply diags out of order: offset 8 before offset 4
	diags := []scan.Diagnostic{
		{Code: "VOID_SCAN", Span: scan.NewSpan(8, 9), Message: "second"},
		{Code: "VOID_SCAN", Span: scan.NewSpan(4, 5), Message: "first"},
	}
	got := report.Render("x.decl", src, diags, report.RenderOptions{})
	firstIdx := strings.Index(got, "first")
	secondIdx := strings.Index(got, "second")
	assert.Less(t, firstIdx, secondIdx, "diagnostics should be sorted by span start")
}
