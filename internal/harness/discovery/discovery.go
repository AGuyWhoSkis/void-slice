// Package discovery is the M12 cascade-hunting harness — a structured
// mutation scanner that exhaustively enumerates single-byte mutations of
// the structural tokens (`{ } ; = [ ]`) over a corpus, then triages the
// post-mutation diagnostics against a small set of anomaly signals.
//
// Unlike the locality harness, which pins known cascade shapes against
// regressions, discovery is a *prospecting* tool. Its output is a
// human-read report (`discovery-report.md`) of findings sorted by signal;
// adoption of any finding into a kanban ticket stays a manual judgment.
//
// Scope of the production package is intentionally narrow: the API to
// enumerate mutations and triage signals lives here, so the build-tagged
// driver (`discovery_test.go`) can stay small and re-runnable from CI
// only when explicitly invoked via `-tags=discovery`.
package discovery

import (
	"bytes"
	"fmt"
	"sort"

	"void-slice/internal/lint"
	"void-slice/internal/scan"
)

// MutationKind distinguishes the structural-token edit shapes: a single-
// byte deletion, a single-byte insertion at a token-pair gap, or a single-
// byte replacement of an existing structural byte with another one from
// the same set. All three are structural-token mutations; the kind drives
// how `Scan` enumerates positions but the downstream signal logic is
// shared.
type MutationKind int

const (
	MutationDelete MutationKind = iota
	MutationInsert
	MutationReplace
)

func (m MutationKind) String() string {
	switch m {
	case MutationDelete:
		return "delete"
	case MutationInsert:
		return "insert"
	case MutationReplace:
		return "replace"
	default:
		return "unknown"
	}
}

// Mutation describes one applied structural-token edit to be paired with
// the diagnostics produced from linting the resulting bytes.
//
// Line is the 1-based line of `Offset` in the *original* source; the
// discovery harness reports findings relative to that origin line so a
// reader can compare against the unedited file.
//
// Token's meaning depends on Kind: for MutationDelete it is the byte that
// was removed (== src[Offset]); for MutationInsert it is the byte
// inserted at Offset; for MutationReplace it is the byte that overwrites
// src[Offset] (and is by construction != src[Offset]).
type Mutation struct {
	Kind   MutationKind
	Token  byte
	Offset int
	Line   int
}

// Signal names the anomaly axis that surfaced a mutation. A single Finding
// may carry more than one signal (e.g. a panic that also looked silent if
// the recovered diag set happened to match baseline).
type Signal string

const (
	SignalSpike  Signal = "diag-count-spike"
	SignalBlowup Signal = "locality-blowup"
	SignalSilent Signal = "no-op-mutation"
	SignalPanic  Signal = "lint-panic"
)

// Finding is one (mutation, post-mutation diagnostics) pair that matched at
// least one signal. Findings are emitted in mutation-enumeration order
// inside Scan; the driver re-sorts globally by severity for the report.
type Finding struct {
	Path     string
	Mutation Mutation
	Diags    []scan.Diagnostic
	Signals  []Signal

	// MaxDelta is the largest |diag.line - Mutation.Line| across post-
	// mutation diagnostics. Populated for every finding (not just
	// Blowup-bearing ones) so the driver can sort blowups by Δ.
	MaxDelta int
}

// SilentMode controls which mutation kinds can fire the Silent signal.
// M12.9's corpus sweep showed inserts dominate Silent-mode noise: an
// insert at a token-pair gap often produces a grammatically-equivalent
// alternate read with identical diags, which is honest but uninteresting
// — not the "linter missed a structural change" shape Silent exists to
// flag. Deletes of `=`, `{`, `}`, etc. are far more likely to be true
// signal. SilentDeletesOnly is the calibrated default; SilentAll restores
// M12.9's original behavior for one-off deep scans.
type SilentMode int

const (
	// SilentDeletesOnly fires Silent only when the mutation is a Delete.
	// Zero-value default — picked deliberately so callers leaving Options
	// at its zero value get the tightened noise floor.
	SilentDeletesOnly SilentMode = iota
	// SilentAll fires Silent for any mutation kind that matches baseline.
	SilentAll
)

// Options control the triage thresholds and supply the baseline diagnostic
// set used by the Silent signal. Zero-value defaults are sensible for
// thresholds; BaseDiags has no default and is required only when the
// caller cares about the Silent signal.
type Options struct {
	KSpike     int               // diag count >= K triggers Spike
	WBlowup    int               // max line-Δ > W triggers Blowup
	BaseDiags  []scan.Diagnostic // baseline lint of src; required for Silent
	SilentMode SilentMode        // gates which mutation kinds can fire Silent

	// MaxMutations caps per-call mutation work. 0 (default) enumerates
	// every structural-byte mutation. K > 0 keeps at most K mutations via
	// deterministic stride sampling (mutations[0], mutations[stride],
	// mutations[2*stride], …) over the concatenated delete+insert list.
	// Stride preserves source order and evenly distributes coverage; it
	// trades exhaustiveness for a linear bound on per-file cost — the
	// natural fix for the harness's intrinsic O(F²) shape on multi-megabyte
	// game files. The driver (`discovery_test.go`) is the calibrated caller
	// and documents the chosen budget; library users with smaller inputs
	// can leave this 0.
	MaxMutations int
}

// Scan applies every structural-token mutation to src and returns the
// findings that match at least one triage signal. Mutation enumeration is
// fully deterministic and proceeds in source order: deletes ascend by
// byte offset, then inserts ascend by (gap-start offset, token-kind).
//
// Lint calls are wrapped in `recover()`; a panicking lint produces a
// Panic-tagged Finding and Scan keeps going.
func Scan(path string, src []byte, opts Options) []Finding {
	if opts.KSpike <= 0 {
		opts.KSpike = 5
	}
	if opts.WBlowup < 0 {
		opts.WBlowup = 0
	}

	toks, _, _ := scan.Scan(src)
	skipMask := buildCommentQuoteMask(src, toks)
	baseSet := makeDiagKeySet(opts.BaseDiags)
	newlines := newlineOffsets(src)

	var mutations []Mutation
	mutations = append(mutations, enumerateDeletes(src, skipMask, newlines)...)
	mutations = append(mutations, enumerateInserts(toks, newlines)...)
	mutations = append(mutations, enumerateReplaces(src, skipMask, newlines)...)
	mutations = sampleMutations(mutations, opts.MaxMutations)

	var findings []Finding
	for _, m := range mutations {
		mutBytes := applyMutation(src, m)
		if bytes.Equal(mutBytes, src) {
			// Defensive: an insert at an out-of-range offset, or a delete
			// of a byte that isn't where we expected, would leave bytes
			// unchanged. Skip — Silent's "actually changed bytes" predicate
			// would gate it anyway and the other signals are meaningless.
			continue
		}
		f, ok := evaluate(path, m, mutBytes, baseSet, newlines, opts)
		if ok {
			findings = append(findings, f)
		}
	}
	return findings
}

// evaluate lints mutBytes and computes signals against the baseline. A
// finding is emitted iff at least one signal fires.
//
// srcNewlines is the original-source newline index. mutBytes never adds or
// removes a `\n` (the structural-byte set excludes it), so we map a mutated
// offset back to its original-source offset rather than re-scanning
// mutBytes for newlines — saving an O(len(src)) pass per mutation.
func evaluate(path string, m Mutation, mutBytes []byte, baseSet map[diagKey]bool, srcNewlines []int, opts Options) (Finding, bool) {
	mutDiags, panicMsg := safeLint(path, mutBytes)

	var signals []Signal
	if panicMsg != "" {
		signals = append(signals, SignalPanic)
	}

	scanDiags := convertLintDiags(mutDiags)
	maxDelta := 0
	for _, d := range scanDiags {
		line := lineAtMutated(srcNewlines, m, d.Span.Start)
		delta := line - m.Line
		if delta < 0 {
			delta = -delta
		}
		if delta > maxDelta {
			maxDelta = delta
		}
	}
	if len(scanDiags) >= opts.KSpike {
		signals = append(signals, SignalSpike)
	}
	if maxDelta > opts.WBlowup {
		signals = append(signals, SignalBlowup)
	}
	if silentAllowed(opts.SilentMode, m.Kind) && diagsMatchBaseline(scanDiags, baseSet) {
		signals = append(signals, SignalSilent)
	}

	if len(signals) == 0 {
		return Finding{}, false
	}
	return Finding{
		Path:     path,
		Mutation: m,
		Diags:    scanDiags,
		Signals:  signals,
		MaxDelta: maxDelta,
	}, true
}

// safeLint wraps lint.New().Lint in a recover so an internal panic surfaces
// as a Finding rather than aborting the harness. Returns the diagnostics
// (possibly nil on panic) and the panic message (empty on success).
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
		// Errors from Lint are non-panic and aren't a discovery signal on
		// their own — record nothing and let the diag-set logic apply.
		return nil, ""
	}
	return out, ""
}

// enumerateDeletes returns one Delete mutation per occurrence of a
// structural byte (`{ } ; = [ ]`) in src that is *not* inside a comment or
// quote span. Ordered by ascending offset.
func enumerateDeletes(src []byte, skipMask []bool, newlines []int) []Mutation {
	var out []Mutation
	for i := 0; i < len(src); i++ {
		if skipMask[i] {
			continue
		}
		if !isStructuralByte(src[i]) {
			continue
		}
		out = append(out, Mutation{
			Kind:   MutationDelete,
			Token:  src[i],
			Offset: i,
			Line:   lineAt(newlines, i),
		})
	}
	return out
}

// enumerateReplaces returns one Replace mutation per (structural-byte
// offset, replacement-byte) where replacement-byte ∈ structuralBytes and
// != src[offset]. Comment- and quote-interior bytes are skipped via the
// shared mask. Ordered by ascending (offset, replacement-byte position in
// structuralBytes). Five replacements per qualifying byte.
func enumerateReplaces(src []byte, skipMask []bool, newlines []int) []Mutation {
	var out []Mutation
	for i := 0; i < len(src); i++ {
		if skipMask[i] {
			continue
		}
		if !isStructuralByte(src[i]) {
			continue
		}
		line := lineAt(newlines, i)
		for _, b := range structuralBytes {
			if b == src[i] {
				continue
			}
			out = append(out, Mutation{
				Kind:   MutationReplace,
				Token:  b,
				Offset: i,
				Line:   line,
			})
		}
	}
	return out
}

// silentAllowed reports whether the Silent signal is permitted for this
// (mode, kind) pair. SilentDeletesOnly suppresses Silent on inserts —
// M12.9's report showed inserts dominate Silent noise. SilentAll lets
// delete and insert fire Silent; replace is gated out under *both* modes.
//
// M12.14 calibration: a sweep that temporarily permitted Silent on
// replace produced 747 findings across 12 files, all noise. The
// distribution splits three ways: 387 came from a 148-diag baseline
// (`gatherdepthminmax.decl`) where the file is already dominated by
// binary-garbage errors and any mutation registers as Silent because
// the pre-existing diag set swamps the swap; 177 came from a 116-diag
// baseline (`arksssblur.decl`) with the same shape; the remaining 183
// were 0-diag baselines (clean grammar files like the animset and
// material decls) where the parser is permissive enough to accept the
// alternate byte in that position. None matched the "linter missed a
// constant-length semantic flip" shape Silent exists to surface, so
// replace stays excluded. Token-level replacement (see the M13 draft)
// is the path to that signal — single-byte structural replace is too
// coarse to distinguish meaning-shift from grammar-permissive.
func silentAllowed(mode SilentMode, kind MutationKind) bool {
	if kind == MutationReplace {
		return false
	}
	if mode == SilentAll {
		return true
	}
	return kind == MutationDelete
}

// sampleMutations applies Options.MaxMutations: if max > 0 and len(in) > max,
// returns every stride-th element where stride = ceil(len(in)/max). Result is
// at most max long, preserves original order, and is deterministic for a
// given (len(in), max). Passing max <= 0 or len(in) <= max returns in.
func sampleMutations(in []Mutation, max int) []Mutation {
	if max <= 0 || len(in) <= max {
		return in
	}
	stride := (len(in) + max - 1) / max
	out := make([]Mutation, 0, max)
	for i := 0; i < len(in); i += stride {
		out = append(out, in[i])
	}
	return out
}

// enumerateInserts returns one Insert mutation per (adjacent-token-pair
// gap, kind) at the gap start (Token[i].End). Zero-width gaps are
// skipped — there is no syntactically meaningful position to insert into.
// Ordered by ascending (gap-start, kind index).
func enumerateInserts(toks []scan.Token, newlines []int) []Mutation {
	if len(toks) < 2 {
		return nil
	}
	var out []Mutation
	for i := 0; i < len(toks)-1; i++ {
		p := toks[i].Span.End
		if p >= toks[i+1].Span.Start {
			continue
		}
		line := lineAt(newlines, p)
		for _, b := range structuralBytes {
			out = append(out, Mutation{
				Kind:   MutationInsert,
				Token:  b,
				Offset: p,
				Line:   line,
			})
		}
	}
	return out
}

// newlineOffsets returns the byte offsets of '\n' in src, ascending. Used
// for line-lookup via binary search — scan.LineIndex.PosAt is linear over
// newlines, which becomes the dominant cost when enumerating thousands of
// mutations on a multi-megabyte file.
func newlineOffsets(src []byte) []int {
	var out []int
	for i, b := range src {
		if b == '\n' {
			out = append(out, i)
		}
	}
	return out
}

// lineAt returns the 1-based line number of offset, computed by binary
// search over newlines. Matches scan.LineIndex.PosAt's line semantics
// (offset == newline-offset stays on the line ending at that newline).
func lineAt(newlines []int, offset int) int {
	lo, hi := 0, len(newlines)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if newlines[mid] < offset {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo + 1
}

// lineAtMutated maps a mutated-source offset back to a line in the
// original source's newline index. Single-byte delete/insert shifts every
// byte after the mutation point by ±1 — adjust the offset, then run the
// usual binary search. Replace preserves length, so the mapping is the
// identity; the case is listed explicitly to keep the switch exhaustive.
func lineAtMutated(srcNewlines []int, m Mutation, mutOffset int) int {
	srcOffset := mutOffset
	switch m.Kind {
	case MutationDelete:
		if mutOffset >= m.Offset {
			srcOffset = mutOffset + 1
		}
	case MutationInsert:
		if mutOffset > m.Offset {
			srcOffset = mutOffset - 1
		}
	case MutationReplace:
		// length-preserving — no offset shift
	}
	return lineAt(srcNewlines, srcOffset)
}

// applyMutation returns the mutated source. Single-byte delete, single-
// byte insert, or single-byte replace; the implementation never aliases
// the input slice.
func applyMutation(src []byte, m Mutation) []byte {
	switch m.Kind {
	case MutationDelete:
		if m.Offset < 0 || m.Offset >= len(src) {
			return src
		}
		out := make([]byte, 0, len(src)-1)
		out = append(out, src[:m.Offset]...)
		out = append(out, src[m.Offset+1:]...)
		return out
	case MutationInsert:
		if m.Offset < 0 || m.Offset > len(src) {
			return src
		}
		out := make([]byte, 0, len(src)+1)
		out = append(out, src[:m.Offset]...)
		out = append(out, m.Token)
		out = append(out, src[m.Offset:]...)
		return out
	case MutationReplace:
		if m.Offset < 0 || m.Offset >= len(src) {
			return src
		}
		out := make([]byte, len(src))
		copy(out, src)
		out[m.Offset] = m.Token
		return out
	default:
		return src
	}
}

// buildCommentQuoteMask returns a per-byte boolean mask marking every
// offset that falls inside a COMMENT_LINE, COMMENT_BLOCK, or
// QUOTE_LITERAL token span. Mutating bytes inside these spans is mostly
// noise — they're string/comment content, not structural punctuation —
// so the harness skips them.
func buildCommentQuoteMask(src []byte, toks []scan.Token) []bool {
	mask := make([]bool, len(src))
	for _, t := range toks {
		switch t.Kind {
		case scan.KindCommentLine, scan.KindCommentBlock, scan.KindQuoteLiteral:
			for i := t.Span.Start; i < t.Span.End && i < len(src); i++ {
				if i >= 0 {
					mask[i] = true
				}
			}
		}
	}
	return mask
}

// structuralBytes is the closed set of single-byte punctuations the
// discovery harness mutates. Mirrors the parser's "tokens that change
// structural meaning"; widening this set is an out-of-scope change.
var structuralBytes = []byte{'{', '}', ';', '=', '[', ']'}

func isStructuralByte(b byte) bool {
	for _, s := range structuralBytes {
		if b == s {
			return true
		}
	}
	return false
}

// diagKey is the equivalence used to compare baseline and post-mutation
// diagnostic sets for the Silent signal. Span equality is literal — the
// spec accepts that mutation-induced byte shifts make Silent rare for
// already-broken files, which is the documented intent.
type diagKey struct {
	Code string
	Span scan.Span
}

func makeDiagKeySet(diags []scan.Diagnostic) map[diagKey]bool {
	out := make(map[diagKey]bool, len(diags))
	for _, d := range diags {
		out[diagKey{Code: string(d.Code), Span: d.Span}] = true
	}
	return out
}

func diagsMatchBaseline(diags []scan.Diagnostic, baseSet map[diagKey]bool) bool {
	if len(diags) != len(baseSet) {
		return false
	}
	for _, d := range diags {
		if !baseSet[diagKey{Code: string(d.Code), Span: d.Span}] {
			return false
		}
	}
	return true
}

// convertLintDiags rehouses lint.Diagnostic in scan.Diagnostic shape — the
// discovery API exposes the latter so callers don't have to depend on the
// `lint` package's Severity wrapper just to read diag positions.
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

// SortFindings orders findings for human triage: panic first, then silent,
// then blowups by Δ desc, then spikes by diag-count desc, then by path
// and offset for deterministic tie-breaking. The driver calls this once
// across the per-file accumulated findings before rendering the report.
func SortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		ar, br := signalRank(a), signalRank(b)
		if ar != br {
			return ar < br
		}
		// Within rank, prefer larger blowup deltas, then larger diag
		// counts — both surface "more broken" findings first.
		if a.MaxDelta != b.MaxDelta {
			return a.MaxDelta > b.MaxDelta
		}
		if len(a.Diags) != len(b.Diags) {
			return len(a.Diags) > len(b.Diags)
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Mutation.Offset < b.Mutation.Offset
	})
}

// signalRank gives the primary-sort key: panic > silent > blowup > spike.
// Findings that bear multiple signals take the strongest one.
func signalRank(f Finding) int {
	hasPanic, hasSilent, hasBlowup := false, false, false
	for _, s := range f.Signals {
		switch s {
		case SignalPanic:
			hasPanic = true
		case SignalSilent:
			hasSilent = true
		case SignalBlowup:
			hasBlowup = true
		}
	}
	switch {
	case hasPanic:
		return 0
	case hasSilent:
		return 1
	case hasBlowup:
		return 2
	default:
		return 3
	}
}
