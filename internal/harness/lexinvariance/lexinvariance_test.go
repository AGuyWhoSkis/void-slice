//go:build lexinvariance

// Package lexinvariance_test (build tag `lexinvariance`) is the driver
// for the lexical-equivalence harness. It is intentionally excluded
// from the default `go test ./...` run — the sweep walks the full
// corpus and emits a triage report rather than a pass/fail gate.
//
// Invocation:
//
//	go test -tags=lexinvariance ./internal/harness/lexinvariance/
//
// The driver walks the same corpus the discovery harness walks
// (testdata/golden + testdata/cascades), baselines each file's lint,
// calls lexinvariance.Scan, then writes lexinvariance-report.md at repo
// root. ~4 kinds × 2 variants per file gives well under a thousand
// lints corpus-wide; expect the sweep to finish in a few seconds.
package lexinvariance_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"void-slice/internal/harness/lexinvariance"
	"void-slice/internal/lint"
)

var corpusRoots = []string{
	filepath.Join("..", "..", "..", "testdata", "golden"),
	filepath.Join("..", "..", "..", "testdata", "cascades"),
}

const reportPath = "../../../lexinvariance-report.md"

func TestLexinvarianceSweep(t *testing.T) {
	files := collectCorpus(t)
	if len(files) == 0 {
		t.Fatal("no corpus files found — check corpusRoots")
	}

	linter := lint.New()
	var allFindings []lexinvariance.Finding
	var totalVariants int

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
		baseScan := convertLintDiags(baseDiags)

		// Count *attempted* variants for the header: kinds × variantsPerKind.
		// `lexinvariance.Scan` skips bytes-unchanged and ok=false variants
		// internally, so the attempted figure is the cleanest "denominator"
		// for divergence rates.
		totalVariants += len(lexinvariance.AllKinds()) * lexinvariance.VariantsPerKind()

		findings := lexinvariance.Scan(path, src, lexinvariance.Options{BaseDiags: baseScan})
		allFindings = append(allFindings, findings...)
	}

	lexinvariance.SortFindings(allFindings)
	report := renderReport(len(files), totalVariants, allFindings)
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	t.Logf("lexinvariance: %d files, %d variants attempted, %d findings → %s",
		len(files), totalVariants, len(allFindings), reportPath)
}

func renderReport(filesScanned, variantsAttempted int, findings []lexinvariance.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Lexinvariance report — %s\n\n", time.Now().UTC().Format(time.RFC3339))

	perKind := map[lexinvariance.TransformKind]int{}
	for _, f := range findings {
		perKind[f.Transform]++
	}

	fmt.Fprintf(&b, "## Corpus\n")
	fmt.Fprintf(&b, "- %d files scanned\n", filesScanned)
	fmt.Fprintf(&b, "- %d variants attempted (kinds × variants × files; per-file skips not subtracted)\n", variantsAttempted)
	fmt.Fprintf(&b, "- %d findings\n", len(findings))
	fmt.Fprintf(&b, "  - reindent:   %d\n", perKind[lexinvariance.TransformReindent])
	fmt.Fprintf(&b, "  - tabspace:   %d\n", perKind[lexinvariance.TransformTabSpace])
	fmt.Fprintf(&b, "  - intertoken: %d\n", perKind[lexinvariance.TransformInterTokenPadding])
	fmt.Fprintf(&b, "  - blankline:  %d\n\n", perKind[lexinvariance.TransformBlankLineJitter])

	fmt.Fprintf(&b, "## Findings\n\n")
	fmt.Fprintf(&b, "Sorted: |CodeDiff| desc, then path, then kind, then variant.\n\n")

	for idx, f := range findings {
		fmt.Fprintf(&b, "### F%d: %s (variant %d) — %s\n",
			idx+1, f.Transform, f.VariantIdx, f.Path)
		fmt.Fprintf(&b, "- Baseline codes (%d): %s\n",
			len(f.BaselineCodes), formatCodeList(f.BaselineCodes))
		fmt.Fprintf(&b, "- Variant codes  (%d): %s\n",
			len(f.VariantCodes), formatCodeList(f.VariantCodes))
		fmt.Fprintf(&b, "- Diff: %s\n", formatDiff(f.CodeDiff))
		if len(f.BaselineDiags) > 0 {
			fmt.Fprintf(&b, "- Baseline sample (first %d):\n", len(f.BaselineDiags))
			for _, d := range f.BaselineDiags {
				fmt.Fprintf(&b, "  - %s %s\n", d.Code, d.Message)
			}
		}
		if len(f.VariantDiags) > 0 {
			fmt.Fprintf(&b, "- Variant sample (first %d):\n", len(f.VariantDiags))
			for _, d := range f.VariantDiags {
				fmt.Fprintf(&b, "  - %s %s\n", d.Code, d.Message)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func formatCodeList(codes []string) string {
	if len(codes) == 0 {
		return "[]"
	}
	// Compact run-length encoding: A ×3, B, C ×2.
	type run struct {
		code  string
		count int
	}
	var runs []run
	for _, c := range codes {
		if n := len(runs); n > 0 && runs[n-1].code == c {
			runs[n-1].count++
			continue
		}
		runs = append(runs, run{c, 1})
	}
	parts := make([]string, 0, len(runs))
	for _, r := range runs {
		if r.count == 1 {
			parts = append(parts, r.code)
		} else {
			parts = append(parts, fmt.Sprintf("%s ×%d", r.code, r.count))
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatDiff(diff []lexinvariance.SignedCode) string {
	if len(diff) == 0 {
		return "(empty)"
	}
	// Run-length encode same-sign-same-code runs so a 148× single-code
	// divergence reads as "-VOID_SCAN ×148" instead of dumping 4KB of
	// repeats. The compaction is per-(sign, code) pair, so a mixed
	// run "+A, -A, +A" stays expanded.
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
	sort.Strings(paths)
	return paths
}

// lintableExt and convertLintDiags live in lexinvariance_unit_test.go
// (no build tag); they're shared between the build-tagged driver, the
// unit tests, and the no-tag hard-gate test in lexinvariance_gate_test.go.
