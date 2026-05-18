package main

// Family: linesplit.
//
// V0 inserts a single '\n' into the first inter-token gap that contains
// at least one space and is not inside a comment or quoted literal. The
// edit is logically a no-op under the M13 inter-token-whitespace
// invariant — both spaces and newlines are inter-token whitespace, so
// the token stream is unchanged. Any diagnostic-multiset diff is a
// blank-line-axis surprise that the M13 gate explicitly parks.
//
// V1 joins the first occurrence of two consecutive newlines back into
// one (deleting one of them). Logically the inverse of V0.

import (
	"bytes"
	"fmt"
	"os"

	"void-slice/internal/scan"
)

func init() { register("linesplit", runLineSplit) }

func runLineSplit(_ []string) {
	paths := walkCorpus("testdata/golden")
	fmt.Printf("# linesplit: %d files\n", len(paths))
	found := 0
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		base := lintCodes(p, src)
		toks, _, _ := scan.Scan(src)

		// V0: insert '\n' into the first qualifying inter-token gap.
		off := firstInterTokenSpace(src, toks)
		if off >= 0 {
			mut := make([]byte, 0, len(src)+1)
			mut = append(mut, src[:off]...)
			mut = append(mut, '\n')
			mut = append(mut, src[off:]...)
			vari := lintCodes(p, mut)
			if !sameMultiset(base, vari) {
				emit(p, "linesplit", 0, off, off+1, diffMultisets(base, vari), "insert-newline-in-gap")
				found++
			}
		}

		// V1: drop one '\n' from the first run of >= 2 consecutive
		// newlines (also a no-op under the invariance axiom).
		idx := bytes.Index(src, []byte("\n\n"))
		if idx >= 0 {
			mut := make([]byte, 0, len(src))
			mut = append(mut, src[:idx]...)
			mut = append(mut, src[idx+1:]...)
			vari := lintCodes(p, mut)
			if !sameMultiset(base, vari) {
				emit(p, "linesplit", 1, idx, idx+1, diffMultisets(base, vari), "remove-one-newline-from-run")
				found++
			}
		}
	}
	fmt.Printf("# linesplit: %d findings\n", found)
}

// firstInterTokenSpace returns the offset of the first ' ' byte that sits
// strictly between two tokens (in the gap toks[i].End..toks[i+1].Start),
// or -1 if no such gap contains a space. The byte's offset is the offset
// at which we will splice a '\n'.
func firstInterTokenSpace(src []byte, toks []scan.Token) int {
	for i := 0; i+1 < len(toks); i++ {
		gs, ge := toks[i].Span.End, toks[i+1].Span.Start
		if gs >= ge {
			continue
		}
		for j := gs; j < ge; j++ {
			if src[j] == ' ' {
				return j + 1
			}
		}
	}
	return -1
}
