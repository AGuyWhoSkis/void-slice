// Package lexinvariance is the M12 lexical-equivalence harness — a
// whitespace/blank-line variation generator that pins the linter against
// the invariance axiom:
//
//	two inputs whose token streams are identical after removing whitespace
//	and comments must produce the same diagnostic-code multiset.
//
// Discovery (M12.9–M12.14) covers structural-byte mutations of
// `{ } ; = [ ]`. Locality (M12.1) covers ±W-line cascade locality.
// Neither sees whitespace-only divergence: locality has no whitespace
// axis, and discovery's mutations are single-byte structural edits —
// two whitespace layouts of the same logical source are not single-byte
// mutations of each other.
//
// Like discovery, this harness is a prospecting tool: it emits a
// human-read report (`lexinvariance-report.md`) sorted by divergence
// size. Adoption of any finding into a fix ticket is manual judgment.
// The package itself files no fixes.
//
// The build-tagged driver (`lexinvariance_test.go`) is excluded from
// the default `go test ./...` run; invoke via:
//
//	go test -tags=lexinvariance ./internal/harness/lexinvariance/
package lexinvariance

import (
	"bytes"
	"fmt"
	"sort"

	"void-slice/internal/lint"
	"void-slice/internal/scan"
)

// TransformKind names the lexical-variation axis. Each kind has a fixed
// number of deterministic variants (currently 2); the (kind, variantIdx)
// pair fully specifies the transform applied.
type TransformKind int

const (
	// TransformReindent rewrites leading-of-line whitespace.
	TransformReindent TransformKind = iota
	// TransformTabSpace rewrites tab/space encoding for whitespace bytes
	// outside comments and quotes.
	TransformTabSpace
	// TransformInterTokenPadding rewrites bytes strictly between adjacent
	// tokens (the gap that the scanner already discards).
	TransformInterTokenPadding
	// TransformBlankLineJitter inserts or removes blank lines between
	// top-level constructs (lines whose first non-whitespace byte sits at
	// column 1).
	TransformBlankLineJitter
)

func (k TransformKind) String() string {
	switch k {
	case TransformReindent:
		return "reindent"
	case TransformTabSpace:
		return "tabspace"
	case TransformInterTokenPadding:
		return "intertoken"
	case TransformBlankLineJitter:
		return "blankline"
	default:
		return "unknown"
	}
}

// variantsPerKind is the fixed count of deterministic variants each kind
// emits. Two is enough to surface both "tighter than source" and "looser
// than source" axes per kind. Composed transforms are out of M12.15 scope.
const variantsPerKind = 2

// AllKinds returns every TransformKind in enumeration order. Exposed so
// the driver can iterate deterministically and callers don't have to
// hard-code the list.
func AllKinds() []TransformKind {
	return []TransformKind{
		TransformReindent,
		TransformTabSpace,
		TransformInterTokenPadding,
		TransformBlankLineJitter,
	}
}

// VariantsPerKind reports the number of variants each kind emits.
func VariantsPerKind() int { return variantsPerKind }

// SignedCode is one entry in a finding's CodeDiff: a diagnostic code and
// the side it surfaced on. Sign +1 means the variant emitted it and the
// baseline didn't; -1 means the baseline emitted it and the variant
// didn't. A finding's CodeDiff is the symmetric difference of the
// baseline and variant code multisets, with multiplicities preserved
// (one entry per unmatched diagnostic occurrence).
type SignedCode struct {
	Code string
	Sign int
}

// Finding is one (transform-variant, divergent-multiset) pair. Emitted
// iff applying the transform to src yields bytes whose lint diagnostic-
// code multiset differs from the baseline's.
//
// BaselineDiags / VariantDiags carry the first few diagnostics from each
// side (sorted by Span.Start) so the report has something concrete to
// show. A finding's "size" — used for sorting — is len(CodeDiff).
type Finding struct {
	Path          string
	Transform     TransformKind
	VariantIdx    int
	BaselineCodes []string // sorted multiset
	VariantCodes  []string // sorted multiset
	CodeDiff      []SignedCode
	BaselineDiags []scan.Diagnostic
	VariantDiags  []scan.Diagnostic
}

// Options control the scan. BaseDiags is required — callers lint src
// once and pass the result so per-variant calls don't re-lint the
// baseline. The harness applies every (kind, variant) deterministically.
type Options struct {
	BaseDiags []scan.Diagnostic
}

// Transform applies a deterministic lexical variation to src.
//
// Returns (mutated, true) when the transform applied non-trivially or
// the caller should still attempt to lint it (in particular: even an
// identity transform returns true if the contract is "the transform
// fired without error"; the caller is expected to filter
// bytes.Equal(mut, src) before linting).
//
// Returns (src, false) when the transform cannot apply (e.g.
// InterTokenPadding on a source with fewer than 2 tokens, or
// BlankLineJitter on a source with no top-level boundaries).
func Transform(src []byte, kind TransformKind, variantIdx int) ([]byte, bool) {
	if variantIdx < 0 || variantIdx >= variantsPerKind {
		return src, false
	}
	switch kind {
	case TransformReindent:
		return transformReindent(src, variantIdx)
	case TransformTabSpace:
		return transformTabSpace(src, variantIdx)
	case TransformInterTokenPadding:
		return transformInterTokenPadding(src, variantIdx)
	case TransformBlankLineJitter:
		return transformBlankLineJitter(src, variantIdx)
	default:
		return src, false
	}
}

// Scan applies every (kind, variant) transform to src and returns a
// Finding for each whose diagnostic-code multiset differs from the
// baseline.
//
// Lint calls are wrapped in `recover()`; a panicking variant still
// produces a Finding (the empty code list on the panicked side becomes
// part of the diff).
func Scan(path string, src []byte, opts Options) []Finding {
	baseCodes := codeMultiset(opts.BaseDiags)
	baseSample := topDiags(opts.BaseDiags, 3)

	var findings []Finding
	for _, k := range AllKinds() {
		for v := 0; v < variantsPerKind; v++ {
			mut, ok := Transform(src, k, v)
			if !ok {
				continue
			}
			if bytes.Equal(mut, src) {
				continue
			}
			varLintDiags, _ := safeLint(path, mut)
			varDiags := convertLintDiags(varLintDiags)
			varCodes := codeMultiset(varDiags)
			if multisetsEqual(baseCodes, varCodes) {
				continue
			}
			findings = append(findings, Finding{
				Path:          path,
				Transform:     k,
				VariantIdx:    v,
				BaselineCodes: baseCodes,
				VariantCodes:  varCodes,
				CodeDiff:      diffMultisets(baseCodes, varCodes),
				BaselineDiags: baseSample,
				VariantDiags:  topDiags(varDiags, 3),
			})
		}
	}
	return findings
}

// SortFindings orders findings for human triage: largest divergence
// first (longer CodeDiff = more code-multiset shift), then by path,
// then by kind, then by variant. Stable so equal-rank findings keep
// their enumeration order.
func SortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if len(a.CodeDiff) != len(b.CodeDiff) {
			return len(a.CodeDiff) > len(b.CodeDiff)
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Transform != b.Transform {
			return a.Transform < b.Transform
		}
		return a.VariantIdx < b.VariantIdx
	})
}

// LexicalTokens returns the byte content of each non-comment token, in
// scan order. Used by the unit-test contract: a valid transform's output
// must produce the same LexicalTokens slice as the input.
//
// Whitespace is already absent from scan.Scan output, so this filter
// removes only COMMENT_LINE and COMMENT_BLOCK — the bytes the invariance
// axiom names as insignificant alongside whitespace.
func LexicalTokens(src []byte) []string {
	toks, _, _ := scan.Scan(src)
	var out []string
	for _, t := range toks {
		switch t.Kind {
		case scan.KindCommentLine, scan.KindCommentBlock:
			continue
		}
		end := t.Span.End
		if end > len(src) {
			end = len(src)
		}
		if t.Span.Start < 0 || t.Span.Start > end {
			continue
		}
		out = append(out, string(src[t.Span.Start:end]))
	}
	return out
}

// ──────────────────────────────────────────────────────────────────────
// Transforms.

// transformReindent rewrites leading-of-line whitespace. V0 swaps each
// leading TAB for 4 spaces (no-op on already-space-indented lines). V1
// strips all leading whitespace, an intentionally aggressive variant
// that exercises the "indentation is insignificant" axiom directly —
// the V1/V2 cascade case from the M12.15 ticket only surfaces under a
// variant that *removes* leading whitespace, not one that re-encodes it.
func transformReindent(src []byte, variantIdx int) ([]byte, bool) {
	if len(src) == 0 {
		return src, false
	}
	toks, _, _ := scan.Scan(src)
	mask := buildCommentQuoteMask(src, toks)

	var out bytes.Buffer
	out.Grow(len(src))
	lineStart := 0
	for i := 0; i <= len(src); i++ {
		atLineEnd := i == len(src) || src[i] == '\n'
		if !atLineEnd {
			continue
		}
		// [lineStart, i) is the line content (no trailing '\n').
		// Find leading whitespace range; bail if it intersects the skip
		// mask (a comment that *starts at column 0* is the realistic case).
		firstNonWS := lineStart
		safe := true
		for firstNonWS < i {
			b := src[firstNonWS]
			if b != ' ' && b != '\t' {
				break
			}
			if mask[firstNonWS] {
				safe = false
				break
			}
			firstNonWS++
		}
		if safe {
			switch variantIdx {
			case 0:
				// tabs → 4 spaces; spaces left as-is.
				for j := lineStart; j < firstNonWS; j++ {
					if src[j] == '\t' {
						out.WriteString("    ")
					} else {
						out.WriteByte(src[j])
					}
				}
			case 1:
				// strip all leading whitespace.
			}
			out.Write(src[firstNonWS:i])
		} else {
			out.Write(src[lineStart:i])
		}
		if i < len(src) {
			out.WriteByte('\n')
		}
		lineStart = i + 1
	}
	return out.Bytes(), true
}

// transformTabSpace rewrites tab/space encoding for whitespace bytes
// outside comments and quotes. V0 expands every TAB to 4 spaces (mid-
// line tabs included — they're inter-token whitespace, which the
// scanner discards). V1 contracts leading runs of exactly 4 spaces into
// a single TAB (limited to leading-of-line; mid-line space runs are
// left alone to keep the contract tight).
func transformTabSpace(src []byte, variantIdx int) ([]byte, bool) {
	if len(src) == 0 {
		return src, false
	}
	toks, _, _ := scan.Scan(src)
	mask := buildCommentQuoteMask(src, toks)

	switch variantIdx {
	case 0:
		var out bytes.Buffer
		out.Grow(len(src))
		for i := 0; i < len(src); i++ {
			if src[i] == '\t' && !mask[i] {
				out.WriteString("    ")
				continue
			}
			out.WriteByte(src[i])
		}
		return out.Bytes(), true
	case 1:
		var out bytes.Buffer
		out.Grow(len(src))
		lineStart := 0
		for i := 0; i <= len(src); i++ {
			atLineEnd := i == len(src) || src[i] == '\n'
			if !atLineEnd {
				continue
			}
			// Rewrite leading whitespace of [lineStart, i): collapse
			// each consecutive run of 4 spaces (where every byte is
			// outside the skip mask) to one '\t'. Tabs and other bytes
			// pass through. Stop at first non-whitespace.
			j := lineStart
			for j < i {
				b := src[j]
				if b == ' ' && j+3 < i &&
					src[j+1] == ' ' && src[j+2] == ' ' && src[j+3] == ' ' &&
					!mask[j] && !mask[j+1] && !mask[j+2] && !mask[j+3] {
					out.WriteByte('\t')
					j += 4
					continue
				}
				if b != ' ' && b != '\t' {
					break
				}
				out.WriteByte(b)
				j++
			}
			out.Write(src[j:i])
			if i < len(src) {
				out.WriteByte('\n')
			}
			lineStart = i + 1
		}
		return out.Bytes(), true
	}
	return src, false
}

// transformInterTokenPadding rewrites the bytes strictly between
// adjacent tokens (where toks[i].End < toks[i+1].Start). V0 collapses
// each gap: one '\n' if the gap contains any '\n', else one ' '. V1
// expands each gap by appending 4 extra spaces immediately before the
// next token's start.
//
// Single-pass forward construction via bytes.Buffer — earlier attempts
// used per-gap `spliceBytes` in reverse order, which is O(F²) on
// multi-MB files with millions of token gaps (each splice copies the
// whole working slice). Forward iteration is O(F) and keeps the corpus
// sweep inside the verification budget.
//
// Edge case: a source with fewer than 2 tokens has no gaps; returns
// (src, false). The driver simply moves on.
func transformInterTokenPadding(src []byte, variantIdx int) ([]byte, bool) {
	toks, _, _ := scan.Scan(src)
	if len(toks) < 2 {
		return src, false
	}

	var out bytes.Buffer
	out.Grow(len(src))

	// Bytes before the first token (typically leading whitespace) — pass
	// through unchanged. They aren't an inter-token gap.
	out.Write(src[:toks[0].Span.Start])

	for i := 0; i < len(toks); i++ {
		tEnd := effectiveTokenEnd(src, toks[i])
		// The token's own bytes.
		out.Write(src[toks[i].Span.Start:tEnd])

		if i == len(toks)-1 {
			// Bytes after the last token — pass through.
			out.Write(src[tEnd:])
			break
		}

		gapStart := tEnd
		gapEnd := toks[i+1].Span.Start
		if gapStart >= gapEnd {
			// Adjacent tokens (zero-width gap). Nothing to do.
			continue
		}

		switch variantIdx {
		case 0:
			if bytes.IndexByte(src[gapStart:gapEnd], '\n') >= 0 {
				out.WriteByte('\n')
			} else {
				out.WriteByte(' ')
			}
		case 1:
			out.Write(src[gapStart:gapEnd])
			out.WriteString("    ")
		}
	}
	return out.Bytes(), true
}

// transformBlankLineJitter inserts or removes blank lines between
// top-level constructs. A "top-level boundary" is a line whose first
// byte is non-whitespace (i.e. the line starts at column 1 with content),
// other than the first such line in the file. V0 inserts one extra '\n'
// immediately before each boundary line. V1 removes one preceding blank
// line at each boundary if there is one.
//
// Single-pass forward construction — see transformInterTokenPadding for
// the same rationale (per-edit splicing would be O(F²) on large files).
func transformBlankLineJitter(src []byte, variantIdx int) ([]byte, bool) {
	if len(src) == 0 {
		return src, false
	}

	// Enumerate boundary line-start offsets in source order. A boundary
	// is a line whose first byte is non-whitespace, *after* at least one
	// earlier such line.
	var boundaries []int
	seenFirst := false
	lineStart := 0
	for i := 0; i <= len(src); i++ {
		if i == len(src) || src[i] == '\n' {
			if lineStart < len(src) {
				b := src[lineStart]
				if b != ' ' && b != '\t' && b != '\n' {
					if seenFirst {
						boundaries = append(boundaries, lineStart)
					}
					seenFirst = true
				}
			}
			lineStart = i + 1
		}
	}
	if len(boundaries) == 0 {
		return src, false
	}

	var out bytes.Buffer
	out.Grow(len(src) + len(boundaries))

	cursor := 0
	edited := false
	for _, bndy := range boundaries {
		switch variantIdx {
		case 0:
			// Copy [cursor, bndy), insert '\n', advance.
			out.Write(src[cursor:bndy])
			out.WriteByte('\n')
			cursor = bndy
			edited = true
		case 1:
			// Drop one preceding blank-line '\n' if present. The "blank
			// line" check: bndy >= 2 && src[bndy-1] == '\n' && src[bndy-2]
			// == '\n'. Drop src[bndy-1].
			if bndy >= 2 && src[bndy-1] == '\n' && src[bndy-2] == '\n' {
				out.Write(src[cursor : bndy-1])
				cursor = bndy
				edited = true
			}
		}
	}
	out.Write(src[cursor:])

	if !edited {
		return src, false
	}
	return out.Bytes(), true
}

// ──────────────────────────────────────────────────────────────────────
// Shared helpers.

// buildCommentQuoteMask returns a per-byte boolean mask marking offsets
// inside COMMENT_LINE, COMMENT_BLOCK, or QUOTE_LITERAL token spans.
// Identical in shape to discovery's mask helper; duplicated rather than
// shared to keep the spike self-contained — a future refactor can pull
// it into a common helper if a third caller appears.
//
// Uses effectiveTokenEnd to include the trailing `/` of `*/`, which the
// scanner omits from COMMENT_BLOCK spans (see internal/scan/scan.go:140).
// Without that adjustment, transforms that operate on "bytes outside
// any token" would treat the comment-closing `/` as fair game and
// silently break the comment when rewritten.
func buildCommentQuoteMask(src []byte, toks []scan.Token) []bool {
	mask := make([]bool, len(src))
	for _, t := range toks {
		switch t.Kind {
		case scan.KindCommentLine, scan.KindCommentBlock, scan.KindQuoteLiteral:
			end := effectiveTokenEnd(src, t)
			for i := t.Span.Start; i < end && i < len(src); i++ {
				if i >= 0 {
					mask[i] = true
				}
			}
		}
	}
	return mask
}

// effectiveTokenEnd returns t.Span.End, plus 1 when t is a COMMENT_BLOCK
// whose Span.End points at a '/' byte (the closing '/' of '*/'). The
// scanner emits COMMENT_BLOCK spans ending one byte short of `*/` — the
// closing '/' is consumed by an outer-loop `i++` but never folded into
// the token span. Any caller that wants to "skip the whole comment" has
// to compensate; this helper centralizes that.
func effectiveTokenEnd(src []byte, t scan.Token) int {
	end := t.Span.End
	if t.Kind == scan.KindCommentBlock && end >= 0 && end < len(src) && src[end] == '/' {
		end++
	}
	return end
}

// safeLint mirrors discovery.safeLint: a recover-wrapped call so a
// panicking lint surfaces as an empty diag list rather than aborting
// the harness. The harness already treats an empty variant code list as
// a divergence when the baseline isn't empty, so a panic naturally
// shows up as a finding.
func safeLint(path string, src []byte) (diags []lint.Diagnostic, panicMsg string) {
	defer func() {
		if r := recover(); r != nil {
			panicMsg = fmt.Sprintf("%v", r)
			diags = nil
		}
	}()
	linter := lint.New()
	out, err := linter.Lint(path, src)
	if err != nil {
		return nil, ""
	}
	return out, ""
}

// convertLintDiags rehouses lint.Diagnostic in scan.Diagnostic — the
// public Finding type uses scan.Diagnostic so callers don't have to
// import `internal/lint` just to read diag positions.
func convertLintDiags(diags []lint.Diagnostic) []scan.Diagnostic {
	out := make([]scan.Diagnostic, len(diags))
	for i, d := range diags {
		out[i] = scan.Diagnostic{
			Code:    scan.DiagnosticCode(d.Code),
			Span:    d.Span,
			Message: d.Message,
		}
	}
	return out
}

// codeMultiset returns the diagnostic codes in sorted order, preserving
// multiplicities. Two diag sets have the same multiset iff their sorted
// code lists are identical — which is the invariance axiom's
// equivalence relation (code only, not span: whitespace shifts span
// offsets without breaking lexical equivalence).
func codeMultiset(diags []scan.Diagnostic) []string {
	out := make([]string, len(diags))
	for i, d := range diags {
		out[i] = string(d.Code)
	}
	sort.Strings(out)
	return out
}

// multisetsEqual reports whether two sorted code multisets are equal.
func multisetsEqual(a, b []string) bool {
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

// diffMultisets returns the symmetric difference of two sorted code
// multisets: codes only in baseline get Sign=-1, codes only in variant
// get Sign=+1. Multiplicities are preserved (one entry per unmatched
// occurrence). Entries are emitted in sorted order, baseline-first.
func diffMultisets(baseline, variant []string) []SignedCode {
	var out []SignedCode
	i, j := 0, 0
	for i < len(baseline) && j < len(variant) {
		switch {
		case baseline[i] < variant[j]:
			out = append(out, SignedCode{Code: baseline[i], Sign: -1})
			i++
		case baseline[i] > variant[j]:
			out = append(out, SignedCode{Code: variant[j], Sign: +1})
			j++
		default:
			i++
			j++
		}
	}
	for ; i < len(baseline); i++ {
		out = append(out, SignedCode{Code: baseline[i], Sign: -1})
	}
	for ; j < len(variant); j++ {
		out = append(out, SignedCode{Code: variant[j], Sign: +1})
	}
	return out
}

// topDiags returns the first n diagnostics by Span.Start. Cheap copy —
// the harness uses this for report previews so we don't store every
// diagnostic on every finding.
func topDiags(diags []scan.Diagnostic, n int) []scan.Diagnostic {
	if len(diags) == 0 {
		return nil
	}
	sorted := make([]scan.Diagnostic, len(diags))
	copy(sorted, diags)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Span.Start < sorted[j].Span.Start
	})
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

