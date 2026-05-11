package discovery_test

import (
	"testing"

	"void-slice/internal/harness/discovery"
	"void-slice/internal/lint"
	"void-slice/internal/scan"
)

// TestSilentModeDeletesOnly pins the M12.11 noise-floor contract: under
// the default SilentMode (SilentDeletesOnly), insert mutations cannot
// fire the Silent signal even when their post-mutation diag set matches
// baseline; deletes still fire normally. SilentAll restores M12.9's
// any-kind behavior.
//
// The fixture is a nested `ident = { … }` shape lifted from the corpus
// (mirrors `cameo_genericcontact..fx.decl`). It lints clean and exercises
// both Silent shapes: `=` deletions collapse to typed-block reads (true
// signal), and several gap-position inserts produce alternate-but-valid
// programs with unchanged diags (noise).
func TestSilentModeDeletesOnly(t *testing.T) {
	src := []byte(`{
	edit = {
		m_groups = {
			num = 1;
			item = {
				m_val = "x";
			}
		}
	}
}
`)

	linter := lint.New()
	baseDiags, err := linter.Lint("test.decl", src)
	if err != nil {
		t.Fatalf("baseline lint: %v", err)
	}
	if len(baseDiags) != 0 {
		t.Fatalf("fixture should lint clean; got %d diags", len(baseDiags))
	}
	var baseScan []scan.Diagnostic // empty baseline — any 0-diag mutation is Silent

	// SilentAll: both kinds may fire.
	allFindings := discovery.Scan("test.decl", src, discovery.Options{
		KSpike:     10000, // disable spike — isolate Silent signal
		WBlowup:    10000, // disable blowup
		BaseDiags:  baseScan,
		SilentMode: discovery.SilentAll,
	})
	allDeletes, allInserts := countSilentByKind(allFindings)
	if allDeletes == 0 {
		t.Fatal("SilentAll: expected silent deletes (the = drops), got 0")
	}
	if allInserts == 0 {
		t.Fatal("SilentAll: expected silent inserts on this fixture, got 0 — fixture no longer exercises insert-silence noise; rebuild it")
	}

	// SilentDeletesOnly (zero-value default): inserts gated, deletes preserved.
	defaultFindings := discovery.Scan("test.decl", src, discovery.Options{
		KSpike:    10000,
		WBlowup:   10000,
		BaseDiags: baseScan,
		// SilentMode left zero-value
	})
	defDeletes, defInserts := countSilentByKind(defaultFindings)
	if defInserts != 0 {
		t.Errorf("SilentDeletesOnly: expected 0 silent inserts, got %d", defInserts)
	}
	if defDeletes != allDeletes {
		t.Errorf("SilentDeletesOnly: silent deletes should match SilentAll; got %d vs %d", defDeletes, allDeletes)
	}
}

func countSilentByKind(findings []discovery.Finding) (deletes, inserts int) {
	for _, f := range findings {
		for _, s := range f.Signals {
			if s != discovery.SignalSilent {
				continue
			}
			if f.Mutation.Kind == discovery.MutationDelete {
				deletes++
			} else {
				inserts++
			}
		}
	}
	return
}
