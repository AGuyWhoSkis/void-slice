package lint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"void-slice/internal/lint"
)

// allowEntry pins the exact number of times {Code, Scope} is allowed to fire
// across the golden corpus. Scope is "*" (any file) or a forward-slash path
// relative to testdata/golden/.
type allowEntry struct {
	Code  string
	Scope string
	Count int
}

const entitiesFile = "d2/game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities"

// drainAllowlist tracks false positives expected to drain over M4.3–M4.8 as
// per-shape grammars and validate rules land. Re-baseline (typically downward)
// when a draining ticket lands. Counts derived from the inventory in
// kanban/goals/M4-decl-taxonomy.md § Allowlist inventory for M4.2.
//
// Drain history:
//   - M4.3 (parse-layer dispatch): PARSE_* from shapes 2/3/4/5/sidecar drained
//     (stub walkers emit no events). Remaining PARSE_UNEXPECTED_TOKEN is
//     Shape-1-only and drains in M4.4. PARSE_EXPECTED_SYMBOL fully drained.
//     VOID_SCAN is scan-layer; unaffected by parse dispatch.
//   - M4.4–M4.7 (Shape 1/2/3/5 walkers + .decl.xml no-op):
//     PARSE_UNEXPECTED_TOKEN drains to zero — Shape-1 walker now accepts
//     bare-`{` top-level + tuple values; Shape-2/5 + Shape-3 walkers replace
//     stubs. Scanner promotes `.` and `/` to SYMBOL, draining Shape-3 + bulk
//     of Shape-4 VOID_SCAN. `.decl.xml` becomes a recognized lint-layer
//     no-op, draining sidecar VOID_SCAN. Residual VOID_SCAN is the Shape-4
//     renderprog carve-out + the EOF unterminated-comment fixture.
//   - M4.8 (validate-rule audit): VALIDATE_ARRAY_COUNT_MISMATCH (×96) and
//     VALIDATE_ARRAY_MISSING_NUM (×51) retired. The corpus revealed `num` is
//     array capacity, not item count — sparse partial overrides of inherited
//     arrays are legal. Inheritance-aware re-introduction tracked in G-B.1.
//     ARRAY_INDEX_OOB and ARRAY_DUP_INDEX survived; both fire zero times on
//     the corpus.
var drainAllowlist = []allowEntry{
	{Code: "VOID_SCAN", Scope: "generated.decls.renderprog.tlf.gatherdepthminmax.decl", Count: 148},
	{Code: "VOID_SCAN", Scope: "generated.decls.renderprog.arksssblur.decl", Count: 116},
	{Code: "VOID_SCAN", Scope: "eof.block-comment-unterminated.decl", Count: 1},
}

// permanentResidual: documented correct behavior. Per the M4.2 ticket,
// LINT_VE_INCONSISTENCY stays out of the drain allowlist — it is correct.
// Counted strictly here so silent regressions still fail.
var permanentResidual = []allowEntry{
	{Code: "LINT_VE_INCONSISTENCY", Scope: entitiesFile, Count: 1},
}

func TestGoldenAllowlist(t *testing.T) {
	corpusRoot := filepath.Join("..", "..", "testdata", "golden")

	expected := append([]allowEntry(nil), drainAllowlist...)
	expected = append(expected, permanentResidual...)

	actual := make([]int, len(expected))
	type unallow struct {
		code, file string
	}
	unallowed := make(map[unallow]int)

	linter := lint.New()

	_ = filepath.WalkDir(corpusRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !lintableExt(d.Name()) {
			return nil
		}

		src, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Errorf("%s: read: %v", path, readErr)
			return nil
		}

		rel, relErr := filepath.Rel(corpusRoot, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		var diags []lint.Diagnostic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s: panic during lint: %v", rel, r)
				}
			}()
			diags, err = linter.Lint(path, src)
		}()
		if err != nil {
			t.Errorf("%s: lint error: %v", rel, err)
			return nil
		}

		for _, dg := range diags {
			if idx := matchEntry(expected, dg.Code, rel); idx >= 0 {
				actual[idx]++
			} else {
				unallowed[unallow{dg.Code, rel}]++
			}
		}
		return nil
	})

	for i, want := range expected {
		got := actual[i]
		if got != want.Count {
			t.Errorf("count mismatch: %s in %s: want %d, got %d (delta %+d)",
				want.Code, want.Scope, want.Count, got, got-want.Count)
		}
	}

	if len(unallowed) > 0 {
		keys := make([]unallow, 0, len(unallowed))
		for k := range unallowed {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].code != keys[j].code {
				return keys[i].code < keys[j].code
			}
			return keys[i].file < keys[j].file
		})
		var b strings.Builder
		fmt.Fprintf(&b, "unallowlisted diagnostics across %d (code, file) pairs:\n", len(keys))
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s × %d at %s\n", k.code, unallowed[k], k.file)
		}
		t.Error(b.String())
	}
}

// lintableExt returns true for the file extensions the lint facade understands,
// including the multi-extension `.decl.xml` sidecar form.
func lintableExt(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".decl.xml") {
		return true
	}
	switch filepath.Ext(lower) {
	case ".decl", ".entitydef", ".entities", ".cfg":
		return true
	}
	return false
}

// matchEntry returns the index of the most-specific allow entry whose code
// matches and whose scope covers `path` (forward-slash, relative to corpus
// root), or -1 if none match. File-scoped entries beat tree-wide ("*").
func matchEntry(entries []allowEntry, code, path string) int {
	best := -1
	for i, e := range entries {
		if e.Code != code {
			continue
		}
		if e.Scope == "*" {
			if best < 0 {
				best = i
			}
			continue
		}
		if e.Scope == path {
			return i
		}
	}
	return best
}
