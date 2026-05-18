package main

// Family: idrename.
//
// For each file, picks the most common IDENTIFIER token and renames every
// occurrence in the file to that identifier with an added `_R` suffix
// (e.g. `inherit` → `inherit_R`). Renames are consistent — every
// occurrence is rewritten the same way — so the lexical-equivalence
// axiom isn't relevant here; this probes whether the linter has any
// identifier-name-dependent semantics that would react to the rename.

import (
	"fmt"
	"os"
	"sort"

	"void-slice/internal/scan"
)

func init() { register("idrename", runIDRename) }

func runIDRename(_ []string) {
	paths := walkCorpus("testdata/golden")
	fmt.Printf("# idrename: %d files\n", len(paths))
	found := 0
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		base := lintCodes(p, src)

		toks, _, _ := scan.Scan(src)
		target := mostCommonIdentifier(src, toks)
		if target == "" {
			continue
		}

		// Build mutated bytes by rewriting every IDENTIFIER token whose
		// content matches `target` into `target + "_R"`. Token order is
		// scan-order; iterate in reverse so offsets don't shift.
		mut := make([]byte, len(src))
		copy(mut, src)
		newName := target + "_R"
		for i := len(toks) - 1; i >= 0; i-- {
			t := toks[i]
			if t.Kind != scan.KindIdentifier {
				continue
			}
			if string(src[t.Span.Start:t.Span.End]) != target {
				continue
			}
			tail := make([]byte, len(mut)-t.Span.End)
			copy(tail, mut[t.Span.End:])
			mut = mut[:t.Span.Start]
			mut = append(mut, []byte(newName)...)
			mut = append(mut, tail...)
		}

		vari := lintCodes(p, mut)
		if sameMultiset(base, vari) {
			continue
		}
		emit(p, "idrename", 0, 0, 0, diffMultisets(base, vari), "rename:"+target+"->"+newName)
		found++
	}
	fmt.Printf("# idrename: %d findings\n", found)
}

// mostCommonIdentifier returns the IDENTIFIER token content with the
// highest occurrence count in src. Ties broken alphabetically. Empty if
// no identifiers in toks.
func mostCommonIdentifier(src []byte, toks []scan.Token) string {
	counts := map[string]int{}
	for _, t := range toks {
		if t.Kind != scan.KindIdentifier {
			continue
		}
		if t.Span.Start < 0 || t.Span.End > len(src) || t.Span.Start >= t.Span.End {
			continue
		}
		counts[string(src[t.Span.Start:t.Span.End])]++
	}
	if len(counts) == 0 {
		return ""
	}
	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if counts[names[i]] != counts[names[j]] {
			return counts[names[i]] > counts[names[j]]
		}
		return names[i] < names[j]
	})
	return names[0]
}
