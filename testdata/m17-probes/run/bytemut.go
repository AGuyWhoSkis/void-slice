package main

// Family: bytemut.
//
// For each file, runs three single-byte mutations that sit on edges
// where small edits commonly happen in real authoring:
//
//   V0 — append-EOF:  append one ' ' byte at end-of-file (an authoring
//                     non-event — adding a trailing space).
//   V1 — append-EOF:  append one '\n' byte at end-of-file (newline-at-EOF
//                     toggle, a real authoring difference).
//   V2 — delete-tail: delete the last byte of the file (commonly an
//                     author's accidental save-without-newline).
//
// Surprise is any diagnostic-multiset diff at all on V0 (a trailing
// space cannot change a token stream), or on V1 if the linter's
// newline-at-EOF posture isn't already explicit and stable.

import (
	"fmt"
	"os"
)

func init() { register("bytemut", runByteMut) }

func runByteMut(_ []string) {
	paths := walkCorpus("testdata/golden")
	fmt.Printf("# bytemut: %d files × 3 variants\n", len(paths))
	found := 0
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		base := lintCodes(p, src)

		// V0: append single space.
		{
			mut := make([]byte, 0, len(src)+1)
			mut = append(mut, src...)
			mut = append(mut, ' ')
			vari := lintCodes(p, mut)
			if !sameMultiset(base, vari) {
				emit(p, "bytemut", 0, len(src), len(src)+1, diffMultisets(base, vari), "append-space")
				found++
			}
		}
		// V1: append single newline.
		{
			mut := make([]byte, 0, len(src)+1)
			mut = append(mut, src...)
			mut = append(mut, '\n')
			vari := lintCodes(p, mut)
			if !sameMultiset(base, vari) {
				emit(p, "bytemut", 1, len(src), len(src)+1, diffMultisets(base, vari), "append-newline")
				found++
			}
		}
		// V2: delete trailing byte (only if file is non-empty).
		if len(src) > 0 {
			mut := make([]byte, len(src)-1)
			copy(mut, src[:len(src)-1])
			vari := lintCodes(p, mut)
			if !sameMultiset(base, vari) {
				emit(p, "bytemut", 2, len(src)-1, len(src), diffMultisets(base, vari), "delete-last-byte")
				found++
			}
		}
	}
	fmt.Printf("# bytemut: %d findings\n", found)
}
