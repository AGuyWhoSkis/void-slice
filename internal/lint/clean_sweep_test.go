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

const (
	entitiesFile        = "d2/game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities"
	entitiesExcerptFile = "maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.excerpt.entities"
)

// permanentAllowlist pins the M4-terminal state of the corpus harness. M4
// closed when the original draining allowlist (M4.2 seed → M4.3–M4.8 churn)
// collapsed to the entries below; see `git log --grep=M4` for the drain
// history. Each entry is justified — silent regressions still fail. Adding a
// new entry must come with a `// Justification:` comment and a ticket.
var permanentAllowlist = []allowEntry{
	// Justification: Shape 4 (renderprog) carve-out per G-C. The no-op walker
	// doesn't lex embedded HLSL, so `# $ < > | @ ? ! & ~ * + . /` bytes inside
	// `hlsl_prefix { … }` surface as VOID_SCAN. Permanent until G-C lands.
	{Code: "VOID_SCAN", Scope: "generated.decls.renderprog.tlf.gatherdepthminmax.decl", Count: 148},
	{Code: "VOID_SCAN", Scope: "generated.decls.renderprog.arksssblur.decl", Count: 116},

	// Justification: intentional malformed fixture. The file exists to pin the
	// scanner's behavior on an unterminated block comment at EOF (M4.1.2).
	{Code: "VOID_SCAN", Scope: "eof.block-comment-unterminated.decl", Count: 1},

	// Justification: documented correct behavior — Void-Explorer inconsistency
	// fires on a real inconsistency in the entitydef corpus. Counted strictly
	// so a regression that masks the true positive still fails.
	{Code: "LINT_VE_INCONSISTENCY", Scope: entitiesFile, Count: 1},
	{Code: "LINT_VE_INCONSISTENCY", Scope: entitiesExcerptFile, Count: 1},
}

func TestGoldenAllowlist(t *testing.T) {
	corpusRoot := filepath.Join("..", "..", "testdata", "golden")

	expected := permanentAllowlist
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
