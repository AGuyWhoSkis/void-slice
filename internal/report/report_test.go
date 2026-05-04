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

	"void-slice/internal/lint"
	"void-slice/internal/report"
	"void-slice/internal/scan"
)

var updateGolden = flag.Bool("update", false, "overwrite golden files with current output")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

// lintFixture runs the full lint facade and converts to []scan.Diagnostic so
// the report layer renders the same diagnostic stream the CLI does. This is
// what unlocks LINT_* codes (binary, VE inconsistency) in the goldens.
func lintFixture(t *testing.T, filename string, src []byte) []scan.Diagnostic {
	t.Helper()
	diags, err := lint.New().Lint(filename, src)
	require.NoError(t, err)
	out := make([]scan.Diagnostic, len(diags))
	for i, d := range diags {
		out[i] = scan.Diagnostic{Code: scan.DiagnosticCode(d.Code), Span: d.Span, Message: d.Message}
	}
	return out
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

func goldenPath(name, ext string) string {
	return filepath.Join("testdata", "golden", name+ext)
}

func checkGolden(t *testing.T, name, ext, got string) {
	t.Helper()
	path := goldenPath(name, ext)
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

// goldenPair pins both pretty (.txt) and JSON (.json) renders for a single
// (name, filename, src, diags) tuple. Use this for any new diagnostic-code
// coverage so the report layer is locked on both output paths.
func goldenPair(t *testing.T, name, filename string, src []byte, diags []scan.Diagnostic) {
	t.Helper()
	pretty := report.Render(filename, src, diags, report.RenderOptions{})
	checkGolden(t, name, ".txt", pretty)
	jsonOut := report.RenderJSON(filename, src, diags)
	checkGolden(t, name, ".json", jsonOut)
}

// --- Golden snapshot tests: broken/.decl fixtures ---

func goldenTest(t *testing.T, fixtureName string) {
	t.Helper()
	src := loadFixture(t, fixtureName)
	filename := fixtureName + ".decl"
	diags := lintFixture(t, filename, src)
	goldenPair(t, fixtureName, filename, src, diags)
}

func TestGolden_DupIndex(t *testing.T)           { goldenTest(t, "dup-index") }
func TestGolden_IndexOOB(t *testing.T)           { goldenTest(t, "index-oob") }
func TestGolden_MissingSemicolon(t *testing.T)   { goldenTest(t, "missing-semicolon") }
func TestGolden_UnterminatedObject(t *testing.T) { goldenTest(t, "unterminated-object") }

// expected-identifier covers PARSE_EXPECTED_IDENTIFIER, PARSE_EXPECTED_SYMBOL,
// and PARSE_UNEXPECTED_TOKEN in a single fixture (a number where a component
// type identifier is required cascades into all three).
func TestGolden_ExpectedIdentifier(t *testing.T) { goldenTest(t, "expected-identifier") }

// --- Corpus-backed VOID_SCAN ---

// VOID_SCAN is pinned against the real corpus fixture for the unterminated
// block-comment EOF case (the only non-renderprog VOID_SCAN in the drain
// allowlist). Real source span; one diagnostic; tiny file.
func TestGolden_VoidScan(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "golden", "eof.block-comment-unterminated.decl")
	src, err := os.ReadFile(path)
	require.NoError(t, err)
	filename := "eof.block-comment-unterminated.decl"
	diags := lintFixture(t, filename, src)
	goldenPair(t, "void-scan", filename, src, diags)
}

// --- Synthetic-backed LINT_* codes ---
//
// LINT_BINARY_FILE and LINT_VE_INCONSISTENCY are produced by the lint facade
// based on filename classification with zero source span. Using the real
// corpus would either pull in a multi-MB .entities file or render NUL bytes
// from binary content into the golden — neither helps. Synthetic source with
// the right extension reproduces the exact production render path.

func TestGolden_LintBinaryFile(t *testing.T) {
	src := []byte("placeholder\n")
	filename := "sample.bwm"
	diags := lintFixture(t, filename, src)
	goldenPair(t, "lint-binary-file", filename, src, diags)
}

func TestGolden_LintVEInconsistency(t *testing.T) {
	src := []byte("Version 1\ncomponent {\n\tcpntTest myT {\n\t\tedit = { m_val = \"x\"; }\n\t}\n}\n")
	filename := "map.entities"
	diags := lintFixture(t, filename, src)
	goldenPair(t, "lint-ve-inconsistency", filename, src, diags)
}

// TestGolden_AllCodesCovered enforces that every diagnostic code emitted by
// the linter has at least one report-layer golden pinning its render. Mirror
// of internal/lint/coverage_audit_test.go::knownCodes — keep in sync. Stops
// future codes from skipping the report-layer surface entirely.
func TestGolden_AllCodesCovered(t *testing.T) {
	expected := []string{
		"PARSE_UNEXPECTED_TOKEN",
		"PARSE_EXPECTED_SYMBOL",
		"PARSE_EXPECTED_IDENTIFIER",
		"PARSE_EXPECTED_SEMICOLON",
		"PARSE_UNTERMINATED_OBJECT",
		"VALIDATE_ARRAY_INDEX_OOB",
		"VALIDATE_ARRAY_DUP_INDEX",
		"LINT_BINARY_FILE",
		"LINT_VE_INCONSISTENCY",
		"VOID_SCAN",
	}

	matches, err := filepath.Glob(filepath.Join("testdata", "golden", "*.txt"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "no goldens found")

	corpus := ""
	for _, p := range matches {
		b, err := os.ReadFile(p)
		require.NoError(t, err)
		corpus += string(b)
	}

	for _, code := range expected {
		assert.Contains(t, corpus, "["+code+"]",
			"no report-layer golden pins %s — add a fixture or update the expected list", code)
	}
}

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
		File        string        `json:"file"`
		Diagnostics []interface{} `json:"diagnostics"`
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

