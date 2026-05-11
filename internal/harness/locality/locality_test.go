package locality_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"void-slice/internal/harness/locality"
)

// corpusRoots are the directories the harness walks. Mirrors
// internal/lint/clean_sweep_test.go's walk: the flat root for broad
// diagnostic-shape coverage plus the spot-checked d2/game1 and doto/game1
// subdirs (the .entities and .decl cells named in §2 of the cascade memo).
var corpusRoots = []string{
	filepath.Join("..", "..", "..", "testdata", "golden"),
	filepath.Join("..", "..", "..", "testdata", "cascades"),
}

// localityWindow is the line-window the property is checked at. W=0 is the
// strict floor: every diagnostic the harness surfaces must intersect the
// mutated line on the nose. M12.6 tightened the parser's EOF anchor for
// the `}\n`-tail shape (mutation leaves `\n\n` at EOF) and re-routed
// PARSE_EXPECTED_SEMICOLON at EOF to the value-token end, which together
// closed the remaining +1-line slop that kept the broad sweep at W=1.
const localityWindow = 0

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

// skipDirs are corpus subtrees the locality harness deliberately ignores.
// Locality's invariant ("any new diagnostic from a one-byte mutation lands
// near mutLine") presumes a clean baseline; fixtures under these paths are
// intentionally-broken cascade samples meant for harness-specific tests
// (e.g. M12.15's lex-invariance corpus). Mutating them stacks a second
// fault onto an existing one and re-anchors cascades to lines other than
// mutLine — which is the fixture's contract, not a regression.
var skipDirs = []string{
	filepath.Join("testdata", "cascades", "lexinvariance"),
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
				for _, skip := range skipDirs {
					if strings.HasSuffix(filepath.ToSlash(path), filepath.ToSlash(skip)) {
						return filepath.SkipDir
					}
				}
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
