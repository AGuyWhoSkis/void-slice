package lexinvariance_test

// TestLexinvarianceHardGate is M13.1's CI gate: any inter-token-gap
// transform (Reindent, TabSpace, InterTokenPadding) that shifts the
// diagnostic-code multiset on a `testdata/golden/` file fails the test.
// The remaining transform (BlankLineJitter) is excluded — newline
// mutation is parked as a separate invariant; the build-tagged sweep
// in lexinvariance_test.go still surfaces it for triage.
//
// A red gate must escalate per M13's "fix-or-router, never skip"
// contract: file an engine-fix or input-boundary-router ticket per
// finding. Do not relax IsHardGate, narrow the corpus, or add a skip
// list to make it green.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"void-slice/internal/harness/lexinvariance"
	"void-slice/internal/lint"
)

func TestLexinvarianceHardGate(t *testing.T) {
	root := filepath.Join("..", "..", "..", "testdata", "golden")

	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !lintableExt(d.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(files) == 0 {
		t.Fatalf("no corpus files under %s — gate would silently pass", root)
	}
	sort.Strings(files)

	linter := lint.New()
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		baseDiags, err := linter.Lint(path, src)
		if err != nil {
			t.Errorf("baseline lint %s: %v", path, err)
			continue
		}
		findings := lexinvariance.Scan(path, src, lexinvariance.Options{
			BaseDiags: convertLintDiags(baseDiags),
		})
		for _, f := range findings {
			if !f.Transform.IsHardGate() {
				continue
			}
			t.Errorf("hard-gate finding: %s\n        transform=%s variant=%d\n        diff=%s",
				f.Path, f.Transform, f.VariantIdx, formatHardGateDiff(f.CodeDiff))
		}
	}
}

// formatHardGateDiff renders a CodeDiff in the same `+CODE ×N, -CODE`
// shape the build-tagged sweep's report uses (run-length encoded per
// (sign, code) pair). Duplicated here rather than exported from the
// sweep driver so the gate test stays independent of the build-tagged
// file.
func formatHardGateDiff(diff []lexinvariance.SignedCode) string {
	if len(diff) == 0 {
		return "(empty)"
	}
	type run struct {
		sign  int
		code  string
		count int
	}
	var runs []run
	for _, d := range diff {
		if n := len(runs); n > 0 && runs[n-1].sign == d.Sign && runs[n-1].code == d.Code {
			runs[n-1].count++
			continue
		}
		runs = append(runs, run{d.Sign, d.Code, 1})
	}
	parts := make([]string, 0, len(runs))
	for _, r := range runs {
		sign := "+"
		if r.sign < 0 {
			sign = "-"
		}
		if r.count == 1 {
			parts = append(parts, sign+r.code)
		} else {
			parts = append(parts, fmt.Sprintf("%s%s ×%d", sign, r.code, r.count))
		}
	}
	return strings.Join(parts, ", ")
}
