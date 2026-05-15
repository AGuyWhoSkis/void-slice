// Command run executes one M17.1 edit-class probe against the void-slice
// corpus and emits machine-readable findings to stdout.
//
// Each probe applies a deterministic transform to every lintable file under
// testdata/golden/, lints the file before and after the edit, and prints
// one finding line per case where the diagnostic-code multiset differs.
//
// Invocation:
//
//	go run ./testdata/m17-probes/run -family=<name>
//
// Available families: blankline, punctuation, linesplit, commenttoggle,
// idrename, bytemut. See per-family files for the exact transform.
//
// Output format (one line per finding):
//
//	<path>	kind=<family>	variant=<v>	span=<start..end>	diff=+CODE,-CODE,...	note=<short>
//
// Lines beginning with '#' are commentary (corpus size, files visited, etc.).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"void-slice/internal/lint"
)

var familyRunners = map[string]func([]string){}

func register(name string, fn func([]string)) { familyRunners[name] = fn }

func main() {
	family := flag.String("family", "", "probe family to run (one of blankline, punctuation, linesplit, commenttoggle, idrename, bytemut)")
	flag.Parse()

	if *family == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./testdata/m17-probes/run -family=<name>")
		fmt.Fprintln(os.Stderr, "families:")
		names := make([]string, 0, len(familyRunners))
		for n := range familyRunners {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintln(os.Stderr, "  "+n)
		}
		os.Exit(2)
	}

	fn, ok := familyRunners[*family]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown family: %q\n", *family)
		os.Exit(2)
	}
	fn(flag.Args())
}

// lintableExts mirrors the lexinvariance harness's filter — the same set
// of extensions the linter actually accepts, plus an explicit skip for
// .decl.xml (Void Explorer metadata, recognized but never linted by shape).
var lintableExts = map[string]bool{
	".decl":      true,
	".entities":  true,
	".entitydef": true,
	".cfg":       true,
}

func walkCorpus(root string) []string {
	var out []string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".decl.xml") {
			return nil
		}
		ext := filepath.Ext(p)
		if !lintableExts[ext] {
			return nil
		}
		out = append(out, p)
		return nil
	})
	sort.Strings(out)
	return out
}

// lintCodes returns the sorted diagnostic-code multiset for src linted as
// filename. A panic surfaces as an empty list (the caller compares
// multisets — an empty variant against a non-empty baseline becomes a
// finding, which is the right signal).
func lintCodes(filename string, src []byte) []string {
	codes := []string{}
	defer func() { _ = recover() }()
	diags, err := lint.New().Lint(filename, src)
	if err != nil {
		return codes
	}
	for _, d := range diags {
		codes = append(codes, d.Code)
	}
	sort.Strings(codes)
	return codes
}

// diffMultisets returns +CODE×N / -CODE×N entries in sorted order. +CODE
// means the variant emitted a code the baseline didn't; -CODE means the
// reverse. Multiplicities are reported as ×N suffixes — cascades produce
// thousands of identical PARSE_UNEXPECTED_TOKEN occurrences and dumping
// one entry per occurrence would balloon findings.txt past usefulness.
func diffMultisets(baseline, variant []string) []string {
	add := map[string]int{}
	sub := map[string]int{}
	i, j := 0, 0
	for i < len(baseline) && j < len(variant) {
		switch {
		case baseline[i] < variant[j]:
			sub[baseline[i]]++
			i++
		case baseline[i] > variant[j]:
			add[variant[j]]++
			j++
		default:
			i++
			j++
		}
	}
	for ; i < len(baseline); i++ {
		sub[baseline[i]]++
	}
	for ; j < len(variant); j++ {
		add[variant[j]]++
	}

	keys := make([]string, 0, len(add)+len(sub))
	seen := map[string]bool{}
	for k := range add {
		if !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	for k := range sub {
		if !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		if n := sub[k]; n > 0 {
			out = append(out, fmt.Sprintf("-%s×%d", k, n))
		}
		if n := add[k]; n > 0 {
			out = append(out, fmt.Sprintf("+%s×%d", k, n))
		}
	}
	return out
}

// emit prints one finding line in the format documented in the package
// comment. Note text is condensed to a single token (no spaces) so the
// output stays grep-friendly.
func emit(path, kind string, variant int, spanStart, spanEnd int, diff []string, note string) {
	fmt.Printf("%s\tkind=%s\tvariant=%d\tspan=%d..%d\tdiff=%s\tnote=%s\n",
		path, kind, variant, spanStart, spanEnd, strings.Join(diff, ","), note)
}

func sameMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
