package lint_test

import (
	"os"
	"path/filepath"
	"testing"

	"void-slice/internal/lint"
)

func TestCoverageAudit(t *testing.T) {
	corpusRoot := filepath.Join("..", "..", "void-files")
	if _, err := os.Stat(corpusRoot); os.IsNotExist(err) {
		t.Skip("void-files corpus not present")
	}

	knownCodes := map[string]bool{
		// T1 — parser (parse_constants.go)
		"PARSE_UNEXPECTED_TOKEN":    true,
		"PARSE_EXPECTED_SYMBOL":     true,
		"PARSE_EXPECTED_IDENTIFIER": true,
		"PARSE_EXPECTED_SEMICOLON":  true,
		"PARSE_UNTERMINATED_OBJECT": true,
		// T2 — validator (validate_constants.go)
		"VALIDATE_ARRAY_COUNT_MISMATCH": true,
		"VALIDATE_ARRAY_INDEX_OOB":      true,
		"VALIDATE_ARRAY_DUP_INDEX":      true,
		"VALIDATE_ARRAY_MISSING_NUM":    true,
		// T4 — lint facade
		"LINT_BINARY_FILE":      true,
		"LINT_VE_INCONSISTENCY": true,
		// scan package (scan_constants.go: Codes.SCAN, Codes.SCAN_STRUCTURE)
		"VOID_SCAN":           true,
		"VOID_SCAN_STRUCTURE": true,
	}

	textExts := map[string]bool{
		".decl":      true,
		".entitydef": true,
		".entities":  true,
		".cfg":       true,
	}

	codeCount := make(map[string]int)
	fileCount := 0

	linter := lint.New()

	walkDir := func(dir string) {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return
		}
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !textExts[filepath.Ext(d.Name())] {
				return nil
			}

			fileCount++
			src, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Logf("skipping %s: %v", path, readErr)
				return nil
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panic while linting %s: %v", path, r)
					}
				}()
				diags, lintErr := linter.Lint(path, src)
				if lintErr != nil {
					t.Errorf("lint error for %s: %v", path, lintErr)
					return
				}
				for _, d := range diags {
					codeCount[d.Code]++
				}
			}()

			return nil
		})
	}

	walkDir(filepath.Join(corpusRoot, "doto", "game1"))
	walkDir(filepath.Join(corpusRoot, "d2", "game1"))

	t.Logf("coverage audit: %d files, %d distinct codes", fileCount, len(codeCount))
	for code, count := range codeCount {
		t.Logf("  %s: %d", code, count)
	}

	for code := range codeCount {
		if !knownCodes[code] {
			t.Errorf("undocumented diagnostic code %q — open a follow-up ticket", code)
		}
	}

	for code := range knownCodes {
		if _, observed := codeCount[code]; !observed {
			t.Logf("known code %q not observed in corpus", code)
		}
	}
}
