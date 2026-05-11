package locality_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"void-slice/internal/harness/locality"
	"void-slice/internal/lint"
	"void-slice/internal/scan"
)

// corpusRoots are the directories the harness walks. Mirrors
// internal/lint/clean_sweep_test.go's walk: the flat root for broad
// diagnostic-shape coverage plus the spot-checked d2/game1 and doto/game1
// subdirs (the .entities and .decl cells named in §2 of the cascade memo).
var corpusRoots = []string{
	filepath.Join("..", "..", "..", "testdata", "golden"),
	filepath.Join("..", "..", "..", "testdata", "cascades"),
}

// localityWindow is the line-window the property is checked at. W=1 is the
// published floor from M12.2 — absorbs the off-by-one EOF-anchor ambiguity
// on files that originally had `}\n` (trailing newline after the close): the
// deletion leaves two trailing newlines, and "the line the close was on" is
// the empty line between them, which collapses to the same position as "line
// after EOF" in the mutated source — the parser can't disambiguate from the
// mutated bytes alone. W=1 still pins every cascade shape from §2 of the
// cascade memo; the bare-`}` (no trailing newline) and committed fixture
// cases both land at W=0 via span-intersect.
const localityWindow = 1

func TestLocalityHarness(t *testing.T) {
	files := collectCorpus(t)
	if len(files) == 0 {
		t.Fatal("no corpus files found — check corpusRoots")
	}

	mutators := locality.Mutators()
	shapes := []string{
		locality.ShapeUnterminatedQuote,
		locality.ShapeUnterminatedBlockComment,
		locality.ShapeUnterminatedBrace,
		locality.ShapeMissingSemicolon,
	}

	var totalApplied, totalSkipped int
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		rel, _ := filepath.Rel(filepath.Join("..", "..", ".."), path)
		for _, shape := range shapes {
			mut := mutators[shape]
			v, applied, err := locality.Check(path, src, shape, mut, localityWindow)
			if err != nil {
				t.Errorf("%s [%s]: %v", rel, shape, err)
				continue
			}
			if !applied {
				totalSkipped++
				continue
			}
			totalApplied++
			if v != nil {
				t.Errorf("%s [%s]:\n%s", rel, shape, v.String())
			}
		}
	}

	t.Logf("locality harness: %d (file, shape) pairs applied, %d skipped",
		totalApplied, totalSkipped)
}

// TestLocalityCommittedFixturesAtW0 pins each committed cascade fixture's
// post-mutation diagnostics at W=0 — the strictest property. The fixtures
// are *already* the mutated form (per M12.1), so we lint them directly and
// assert every diagnostic lands within ±0 lines of the expected mutation
// line via span-intersect. Broader corpus sweep at W=1 lives in
// TestLocalityHarness; W=0 on corpus carries the `}\n`-tail anchor ambiguity
// described there.
func TestLocalityCommittedFixturesAtW0(t *testing.T) {
	cases := []struct {
		path    string
		mutLine int
	}{
		// `"x` on line 2; mutation = deleted closing `"`.
		{"unterminated-quote/minimal.decl", 2},
		// `/* unterminated` on line 2; mutation = `*/` never written.
		{"unterminated-block-comment/minimal.decl", 2},
		// File ends after `b = 1;\n` (line 3) — missing `}` would occupy
		// line 4 (the empty line just past content).
		{"unterminated-brace/minimal.decl", 4},
		// `item[1] = { m_val = "c";` on line 9 — missing `}` between the
		// value body and the next `}` (line 10). Pre-M12.7 the diagnostic
		// anchors at EOF (line 14) via greedy-match cascade; the fix re-
		// anchors it on line 9's `{`.
		{"missing-mid-file-brace/minimal.decl", 9},
	}
	linter := lint.New()
	for _, tc := range cases {
		t.Run(filepath.Dir(tc.path), func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "testdata", "cascades", tc.path)
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			diags, err := linter.Lint(path, src)
			if err != nil {
				t.Fatalf("Lint: %v", err)
			}
			li := scan.BuildLineIndex(src)
			for _, d := range diags {
				start := li.PosAt(d.Span.Start)
				end := li.PosAt(d.Span.End)
				startLine := start.Line
				endLine := end.Line
				if endLine < startLine {
					endLine = startLine
				}
				// in-window at W=0: span covers mutLine
				if endLine < tc.mutLine || startLine > tc.mutLine {
					t.Errorf("W=0 violation: %s at %d:%d (span %d..%d) — mutLine=%d, message=%q",
						d.Code, start.Line, start.Col, startLine, endLine, tc.mutLine, d.Message)
				}
			}
		})
	}
}

// collectCorpus walks corpusRoots and returns paths to text-extension files
// the lint facade understands. Mirrors lintableExt in clean_sweep_test.go.
func collectCorpus(t *testing.T) []string {
	t.Helper()
	var paths []string
	for _, root := range corpusRoots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !lintableExt(d.Name()) {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
	}
	return paths
}

func lintableExt(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".decl.xml") {
		// Sidecar XML — lint facade no-ops; harness skips too (mutating
		// them is meaningless).
		return false
	}
	switch filepath.Ext(lower) {
	case ".decl", ".entitydef", ".entities", ".cfg":
		return true
	}
	return false
}
