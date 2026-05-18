package main

// Family: blankline.
//
// Drives lexinvariance.TransformBlankLineJitter (triage-only today, no
// hard gate) across the corpus. V0 inserts one blank line at each top-
// level boundary; V1 removes one blank line preceding each boundary if
// present.
//
// This probe is a thin shim over the existing harness — the transform is
// the production one, only the runner here is throwaway.

import (
	"fmt"
	"os"

	"void-slice/internal/harness/lexinvariance"
)

func init() { register("blankline", runBlankLine) }

func runBlankLine(_ []string) {
	paths := walkCorpus("testdata/golden")
	fmt.Printf("# blankline: %d files\n", len(paths))
	found := 0
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, v := range []int{0, 1} {
			mut, ok := lexinvariance.Transform(src, lexinvariance.TransformBlankLineJitter, v)
			if !ok {
				continue
			}
			if string(mut) == string(src) {
				continue
			}
			base := lintCodes(p, src)
			vari := lintCodes(p, mut)
			if sameMultiset(base, vari) {
				continue
			}
			diff := diffMultisets(base, vari)
			label := "insert-blank"
			if v == 1 {
				label = "remove-blank"
			}
			emit(p, "blankline", v, 0, 0, diff, label)
			found++
		}
	}
	fmt.Printf("# blankline: %d findings\n", found)
}
