package discovery_test

import (
	"testing"

	"void-slice/internal/harness/discovery"
)

// TestReplaceMutationsAppearAfterDeletesAndInserts pins M12.13's
// enumeration ordering: deletes ascend by offset, then inserts at gap
// positions, then replaces. Stride-sampling preserves order, so this
// invariant is what lets the per-file budget bound work uniformly without
// starving any one kind.
func TestReplaceMutationsAppearAfterDeletesAndInserts(t *testing.T) {
	// Two structural bytes (`=`, `;`) → 2 deletes; one token gap → 6
	// inserts; 2 replaces × 5 swaps each = 10 replaces. Trailing newline
	// keeps the IDENT-NUMBER pair from running together.
	src := []byte("a = 1;\n")

	findings := discovery.Scan("test.decl", src, discovery.Options{
		KSpike:     1, // surface every mutation that emits any diag
		WBlowup:    0,
		BaseDiags:  nil,
		SilentMode: discovery.SilentAll,
	})

	kindsSeen := map[discovery.MutationKind]bool{}
	var lastRankSeen int
	rank := func(k discovery.MutationKind) int {
		switch k {
		case discovery.MutationDelete:
			return 0
		case discovery.MutationInsert:
			return 1
		case discovery.MutationReplace:
			return 2
		}
		return -1
	}
	// Findings come back in mutation-enumeration order from Scan (the
	// driver re-sorts globally, but the production API does not).
	for _, f := range findings {
		k := f.Mutation.Kind
		kindsSeen[k] = true
		r := rank(k)
		if r < lastRankSeen {
			t.Errorf("kind %s emitted after a later-ranked kind; ordering broken", k)
		}
		if r > lastRankSeen {
			lastRankSeen = r
		}
	}
	if !kindsSeen[discovery.MutationReplace] {
		t.Fatal("expected at least one MutationReplace finding from `a = 1;`; got none")
	}
}

// TestReplaceProducesDiagCountChange pins the axiom from M12.13: replacing
// a structural byte with another structural byte should shift diagnostics
// in some way the harness can flag. Concretely, `foo = 1;` lints clean,
// but `foo ; 1;` (= → ;) introduces a meaning shift the linter notices.
func TestReplaceProducesDiagCountChange(t *testing.T) {
	src := []byte("foo = 1;\n")

	findings := discovery.Scan("test.decl", src, discovery.Options{
		KSpike:    1, // any diag is a Spike on this 0-diag baseline
		WBlowup:   0,
		BaseDiags: nil,
	})

	var sawReplaceWithDiagShift bool
	for _, f := range findings {
		if f.Mutation.Kind != discovery.MutationReplace {
			continue
		}
		if f.Mutation.Token == ';' && byteAt(src, f.Mutation.Offset) == '=' {
			// the `=` → `;` swap should produce diagnostics relative to a
			// clean baseline
			if len(f.Diags) == 0 {
				t.Errorf("replace `=` → `;` produced 0 diags; expected at least one")
			}
			sawReplaceWithDiagShift = true
			break
		}
	}
	if !sawReplaceWithDiagShift {
		t.Fatal("did not see a `=` → `;` replace finding; enumeration or sampling regressed")
	}
}

// TestReplaceCannotFireSilent pins the M12.14 calibration: silentAllowed
// must return false for MutationReplace under either SilentMode. The
// sweep that temporarily permitted replace-Silent surfaced 747 findings
// across 12 files, all noise — either the file's baseline diag set
// already dominated and any further mutation registered as Silent, or
// the grammar was permissive enough that the alternate byte produced
// identical clean diags. Keep replace excluded; revisit at the token
// level (see the M13 draft) rather than at single-byte structural shape.
//
// Delete-Silent's continued behavior under the chosen default is pinned
// separately by TestSilentModeDeletesOnly — kept disjoint so a failure
// here points precisely at replace-gate regression.
func TestReplaceCannotFireSilent(t *testing.T) {
	// Permissive shape — `[` and `{` are both accepted as the opener of a
	// list-or-object value in many positions, so the parser tends to emit
	// identical diags either way. Exact equivalence is not what's pinned
	// here; the point is that the harness *would* see baseline-matching
	// diags from at least one replace and must still gate it.
	src := []byte(`{
	a = 1;
}
`)

	for _, mode := range []discovery.SilentMode{
		discovery.SilentDeletesOnly,
		discovery.SilentAll,
	} {
		findings := discovery.Scan("test.decl", src, discovery.Options{
			KSpike:     10000, // disable spike
			WBlowup:    10000, // disable blowup
			BaseDiags:  nil,
			SilentMode: mode,
		})
		for _, f := range findings {
			if f.Mutation.Kind != discovery.MutationReplace {
				continue
			}
			for _, s := range f.Signals {
				if s == discovery.SignalSilent {
					t.Errorf("mode=%v: MutationReplace fired SignalSilent at offset %d (replace=%q); M12.14 gate regressed",
						mode, f.Mutation.Offset, f.Mutation.Token)
				}
			}
		}
	}
}

// TestReplaceLineMapping pins that lineAtMutated treats replace as
// length-preserving: a diagnostic at original-source offset O on line L
// should still map to line L after a replace anywhere in the file. The
// fixture spans multiple lines so a misapplied ±1 offset shift would
// surface as a wrong Mutation.Line attribution.
func TestReplaceLineMapping(t *testing.T) {
	// Replacing `=` on line 2 with `;` produces a diag whose origin line
	// should be 2; if lineAtMutated wrongly shifted by 1, we'd see 1 or 3.
	src := []byte("// header\nfoo = 1;\nbar = 2;\n")

	findings := discovery.Scan("test.decl", src, discovery.Options{
		KSpike:    1,
		WBlowup:   0,
		BaseDiags: nil,
	})

	var checked int
	for _, f := range findings {
		if f.Mutation.Kind != discovery.MutationReplace {
			continue
		}
		// Origin line is computed from src's newline index against
		// f.Mutation.Offset — must match the live src.
		want := lineOf(src, f.Mutation.Offset)
		if f.Mutation.Line != want {
			t.Errorf("replace at offset %d: Mutation.Line=%d want=%d", f.Mutation.Offset, f.Mutation.Line, want)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no replace findings observed; cannot validate line mapping")
	}
}

func byteAt(src []byte, off int) byte {
	if off < 0 || off >= len(src) {
		return 0
	}
	return src[off]
}

func lineOf(src []byte, off int) int {
	line := 1
	if off > len(src) {
		off = len(src)
	}
	for i := 0; i < off; i++ {
		if src[i] == '\n' {
			line++
		}
	}
	return line
}
