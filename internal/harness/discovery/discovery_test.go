//go:build discovery

// Package discovery_test (build tag `discovery`) is the driver for the
// structured-mutation scanner. It is intentionally excluded from the
// default `go test ./...` run because a corpus-wide sweep is slow and the
// signal is for human triage rather than CI gating.
//
// Invocation:
//
//	go test -tags=discovery ./internal/harness/discovery/
//
// The driver walks the same corpus as the locality harness, lints each
// file to get baseline diagnostics, calls discovery.Scan, then writes a
// markdown report to `discovery-report.md` at repo root.
package discovery_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"void-slice/internal/harness/discovery"
	"void-slice/internal/lint"
	"void-slice/internal/scan"
)

var corpusRoots = []string{
	filepath.Join("..", "..", "..", "testdata", "golden"),
	filepath.Join("..", "..", "..", "testdata", "cascades"),
}

const reportPath = "../../../discovery-report.md"

// perFileMutationBudget bounds per-file work so the corpus sweep finishes
// inside the verification window. Discovery's intrinsic cost is O(F²):
// mutations grow ∝ F and the linter inside each mutation costs ∝ F. The
// measured lint rate on this corpus is ~13 µs/KB, so the 7.7 MB
// `dunwall_escape_tower_p.entities` lints at ~100 ms — unbounded
// enumeration over its ~5 M structural mutations would take ~140 hours.
// 2000 caps that file at ~3 min and the full corpus at ~5–6 min, well
// inside the ticketed 10-min budget. Files whose enumerated mutation
// count exceeds the budget are stride-sampled (deterministic, source-
// order preserving) by discovery.Scan; smaller files run exhaustively.
//
// Setting this to 0 disables the cap and restores M12.9's exhaustive
// behavior — useful for one-off deep scans but not viable corpus-wide.
const perFileMutationBudget = 2000

func TestDiscoverySweep(t *testing.T) {
	files := collectCorpus(t)
	if len(files) == 0 {
		t.Fatal("no corpus files found — check corpusRoots")
	}

	linter := lint.New()
	var allFindings []discovery.Finding
	var totalEnumerated, totalSampled int
	var totalDeletes, totalInserts, totalReplaces int
	var sampledFiles []sampledFile

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

		opts := discovery.Options{
			KSpike:       5,
			WBlowup:      2,
			BaseDiags:    baseScan,
			MaxMutations: perFileMutationBudget,
		}
		findings := discovery.Scan(path, src, opts)
		allFindings = append(allFindings, findings...)

		d, i, r := countMutations(src)
		totalDeletes += d
		totalInserts += i
		totalReplaces += r
		enumerated := d + i + r
		sampled := enumerated
		if perFileMutationBudget > 0 && enumerated > perFileMutationBudget {
			sampled = sampledCount(enumerated, perFileMutationBudget)
			sampledFiles = append(sampledFiles, sampledFile{
				Path:       path,
				Bytes:      len(src),
				Enumerated: enumerated,
				Sampled:    sampled,
			})
		}
		totalEnumerated += enumerated
		totalSampled += sampled
	}

	discovery.SortFindings(allFindings)
	report := renderReport(len(files), totalDeletes, totalInserts, totalReplaces, totalEnumerated, totalSampled, sampledFiles, allFindings)
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	t.Logf("discovery: %d files, %d enumerated → %d sampled (%d files capped at budget=%d), %d findings → %s",
		len(files), totalEnumerated, totalSampled, len(sampledFiles), perFileMutationBudget, len(allFindings), reportPath)
}

// sampledFile holds the per-file sampling stats the report surfaces so a
// reader can see which files ran exhaustive vs. budgeted.
type sampledFile struct {
	Path       string
	Bytes      int
	Enumerated int
	Sampled    int
}

// sampledCount mirrors discovery.sampleMutations's stride math so the
// report can show exact post-sampling counts without re-running Scan.
func sampledCount(n, max int) int {
	if max <= 0 || n <= max {
		return n
	}
	stride := (n + max - 1) / max
	return (n + stride - 1) / stride
}

// countMutations returns (deletes, inserts, replaces) the harness would
// have produced for src. The exact numbers are for the report header —
// the production `Scan` already does this work internally, but exposing
// it here would require a separate enumeration API; an extra pass over
// src is cheap. Replace yields 5 mutations per structural byte (six in
// the set minus the one already present).
func countMutations(src []byte) (int, int, int) {
	toks, _, _ := scan.Scan(src)
	mask := make([]bool, len(src))
	for _, t := range toks {
		switch t.Kind {
		case scan.KindCommentLine, scan.KindCommentBlock, scan.KindQuoteLiteral:
			for i := t.Span.Start; i < t.Span.End && i < len(src); i++ {
				if i >= 0 {
					mask[i] = true
				}
			}
		}
	}
	var deletes int
	for i := 0; i < len(src); i++ {
		if mask[i] {
			continue
		}
		switch src[i] {
		case '{', '}', ';', '=', '[', ']':
			deletes++
		}
	}
	var inserts int
	for i := 0; i < len(toks)-1; i++ {
		if toks[i].Span.End < toks[i+1].Span.Start {
			inserts += 6
		}
	}
	replaces := deletes * 5
	return deletes, inserts, replaces
}

func renderReport(filesScanned, totalDeletes, totalInserts, totalReplaces, totalEnumerated, totalSampled int, sampledFiles []sampledFile, findings []discovery.Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Discovery report — %s\n\n", time.Now().UTC().Format(time.RFC3339))

	signalCounts := map[discovery.Signal]int{}
	for _, f := range findings {
		for _, s := range f.Signals {
			signalCounts[s]++
		}
	}

	fmt.Fprintf(&b, "## Corpus\n")
	fmt.Fprintf(&b, "- %d files scanned\n", filesScanned)
	fmt.Fprintf(&b, "- %d mutations enumerated (%d deletes, %d inserts, %d replaces)\n",
		totalEnumerated, totalDeletes, totalInserts, totalReplaces)
	fmt.Fprintf(&b, "- %d mutations applied after per-file budget=%d (%d files stride-sampled — see Sampled section)\n",
		totalSampled, perFileMutationBudget, len(sampledFiles))
	fmt.Fprintf(&b, "- %d findings (%d panics, %d silent, %d spikes, %d blowups; a finding can carry multiple signals)\n\n",
		len(findings),
		signalCounts[discovery.SignalPanic],
		signalCounts[discovery.SignalSilent],
		signalCounts[discovery.SignalSpike],
		signalCounts[discovery.SignalBlowup],
	)

	if len(sampledFiles) > 0 {
		fmt.Fprintf(&b, "## Sampled (mutation budget = %d)\n\n", perFileMutationBudget)
		fmt.Fprintf(&b, "Files whose enumerated mutation count exceeded the budget are stride-sampled deterministically.\n\n")
		// Sort by enumerated desc so the heaviest files surface first.
		sortedSamples := make([]sampledFile, len(sampledFiles))
		copy(sortedSamples, sampledFiles)
		sort.SliceStable(sortedSamples, func(i, j int) bool {
			return sortedSamples[i].Enumerated > sortedSamples[j].Enumerated
		})
		for _, s := range sortedSamples {
			fmt.Fprintf(&b, "- %s (%d bytes): %d enumerated → %d sampled\n",
				s.Path, s.Bytes, s.Enumerated, s.Sampled)
		}
		b.WriteByte('\n')
	}

	fmt.Fprintf(&b, "## Findings\n\n")
	fmt.Fprintf(&b, "Sorted: panic > silent > blowup-Δ desc > spike-count desc.\n\n")

	for idx, f := range findings {
		fmt.Fprintf(&b, "### F%d: %s — %s\n", idx+1, headlineSignal(f), f.Path)
		fmt.Fprintf(&b, "- Mutation: %s `%s` at offset %d (line %d)\n",
			f.Mutation.Kind, formatByte(f.Mutation.Token), f.Mutation.Offset, f.Mutation.Line)
		fmt.Fprintf(&b, "- Signals: %s\n", joinSignals(f.Signals))
		fmt.Fprintf(&b, "- Diagnostics: %d (MaxΔ=%d)\n", len(f.Diags), f.MaxDelta)
		if len(f.Diags) > 0 {
			fmt.Fprintf(&b, "- Sample (first %d):\n", min(3, len(f.Diags)))
			sampleDiags := sortedDiagsByLine(f.Diags)
			for i, d := range sampleDiags {
				if i >= 3 {
					break
				}
				fmt.Fprintf(&b, "  - %s %s\n", d.Code, d.Message)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func headlineSignal(f discovery.Finding) string {
	for _, want := range []discovery.Signal{
		discovery.SignalPanic, discovery.SignalSilent, discovery.SignalBlowup, discovery.SignalSpike,
	} {
		for _, s := range f.Signals {
			if s == want {
				if want == discovery.SignalBlowup {
					return fmt.Sprintf("blowup (Δ=%d)", f.MaxDelta)
				}
				if want == discovery.SignalSpike {
					return fmt.Sprintf("spike (n=%d)", len(f.Diags))
				}
				return string(want)
			}
		}
	}
	return "untagged"
}

func joinSignals(signals []discovery.Signal) string {
	parts := make([]string, len(signals))
	for i, s := range signals {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}

func formatByte(b byte) string {
	return string(b)
}

func sortedDiagsByLine(diags []scan.Diagnostic) []scan.Diagnostic {
	out := make([]scan.Diagnostic, len(diags))
	copy(out, diags)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Span.Start < out[j].Span.Start
	})
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func convertLintDiags(diags []lint.Diagnostic) []scan.Diagnostic {
	out := make([]scan.Diagnostic, len(diags))
	for i, d := range diags {
		out[i] = scan.Diagnostic{
			Code:    scan.DiagnosticCode(d.Code),
			Span:    d.Span,
			Message: d.Message,
		}
	}
	return out
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
	return paths
}

func lintableExt(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".decl.xml") {
		return false
	}
	switch filepath.Ext(lower) {
	case ".decl", ".entitydef", ".entities", ".cfg":
		return true
	}
	return false
}
