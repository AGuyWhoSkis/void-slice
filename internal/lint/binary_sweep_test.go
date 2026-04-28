package lint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"void-slice/internal/lint"
)

func TestBinarySweep(t *testing.T) {
	corpusRoot := filepath.Join("..", "..", "void-files")
	if _, err := os.Stat(corpusRoot); os.IsNotExist(err) {
		t.Skip("void-files corpus not present")
	}

	binaryExts := map[string]bool{
		".bwm":               true,
		".tome":              true,
		".navmesh":           true,
		".mapresources":      true,
		".soundpropa":        true,
		".bnavmesh":          true,
		".bphysworld":        true,
		".maprscreusechunk0": true,
	}

	var files []string
	err := filepath.WalkDir(corpusRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if binaryExts[ext] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}

	if len(files) == 0 {
		t.Skip("no binary-extension files in corpus — skipping binary sweep")
	}

	lintFile := func(filename string, src []byte) (diags []lint.Diagnostic, panicVal any) {
		defer func() {
			if r := recover(); r != nil {
				panicVal = r
			}
		}()
		diags, _ = lint.New().Lint(filename, src)
		return diags, nil
	}

	count := 0
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: ReadFile failed: %v", path, err)
			continue
		}

		diags, panicVal := lintFile(path, src)
		if panicVal != nil {
			t.Errorf("%s: Lint panicked: %v", path, panicVal)
			continue
		}

		if len(diags) != 1 {
			t.Errorf("%s: expected 1 diagnostic, got %d: %v", path, len(diags), fmt.Sprintf("%+v", diags))
			count++
			continue
		}
		if diags[0].Severity != lint.Error {
			t.Errorf("%s: expected Severity=Error (%d), got %d", path, lint.Error, diags[0].Severity)
		}
		if diags[0].Code != "LINT_BINARY_FILE" {
			t.Errorf("%s: expected Code=%q, got %q", path, "LINT_BINARY_FILE", diags[0].Code)
		}
		count++
	}

	t.Logf("binary sweep: checked %d files", count)
}
