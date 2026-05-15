package main

// Family: punctuation.
//
// For each lintable corpus file, flips the first occurrence of each of
// three single-byte punctuation classes at a span the scanner returned
// as a SYMBOL token:
//
//	{ → (   (opens object → opens what scanner reads as paren)
//	; → ,   (statement terminator → list separator)
//	" → '   (quote-literal delimiter swap)
//
// One mutation per (file, class). The transform is the minimal possible
// edit — a single byte swap at a known token position. Surprise is any
// diagnostic-multiset diff whose magnitude exceeds 1 (a single byte
// flip producing more than one diagnostic-code change anywhere in the
// file).

import (
	"fmt"
	"os"

	"void-slice/internal/scan"
)

func init() { register("punctuation", runPunctuation) }

type punctSwap struct {
	from, to byte
	label    string
}

var punctSwaps = []punctSwap{
	{'{', '(', "brace-to-paren"},
	{';', ',', "semi-to-comma"},
	{'"', '\'', "dquote-to-squote"},
}

func runPunctuation(_ []string) {
	paths := walkCorpus("testdata/golden")
	fmt.Printf("# punctuation: %d files × %d swaps\n", len(paths), len(punctSwaps))
	found := 0
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		toks, _, _ := scan.Scan(src)
		base := lintCodes(p, src)
		for _, sw := range punctSwaps {
			off := firstSymbolByte(src, toks, sw.from)
			if off < 0 {
				continue
			}
			mut := make([]byte, len(src))
			copy(mut, src)
			mut[off] = sw.to
			vari := lintCodes(p, mut)
			if sameMultiset(base, vari) {
				continue
			}
			diff := diffMultisets(base, vari)
			emit(p, "punctuation", 0, off, off+1, diff, sw.label)
			found++
		}
	}
	fmt.Printf("# punctuation: %d findings\n", found)
}

// firstSymbolByte returns the first byte offset where src[off] == b and
// the byte sits at the start of a scanner-recognised SYMBOL or
// QUOTE_LITERAL token. Returns -1 if no such offset exists. For " the
// QUOTE_LITERAL kind is preferred; for { and ; the SYMBOL kind.
func firstSymbolByte(src []byte, toks []scan.Token, b byte) int {
	for _, t := range toks {
		if t.Span.Start >= len(src) || t.Span.Start < 0 {
			continue
		}
		if src[t.Span.Start] != b {
			continue
		}
		switch b {
		case '"':
			if t.Kind == scan.KindQuoteLiteral {
				return t.Span.Start
			}
		default:
			if t.Kind == scan.KindSymbol {
				return t.Span.Start
			}
		}
	}
	return -1
}
