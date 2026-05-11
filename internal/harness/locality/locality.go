// Package locality is the M12 soundness harness — a property-based locality
// fuzzer that pins the linter against cascading-diagnostic regressions.
//
// For each clean golden file the harness applies a single byte-level mutation
// at a known span and asserts that every new diagnostic (post-baseline) lands
// within ±W lines of the mutated span. The property catches the cascade
// shapes characterized in kanban/goals/M12-cascade-investigation.md §2:
// unterminated-quote, unterminated-block-comment, unterminated-brace. The
// missing-semicolon mutator is the negative case — it must produce no
// outside-line diagnostics, ever.
//
// The harness lives in a Go test (locality_test.go) so it runs alongside
// `go test ./...`. The mutators and predicate are exported here so a future
// fix-ticket implementer can run them ad-hoc against new fixtures without
// re-rigging the throwaway from M12.1.
package locality

import (
	"bytes"
	"fmt"
	"sort"

	"void-slice/internal/lint"
	"void-slice/internal/scan"
)

// Mutator transforms a clean source into a corrupted one. Returns the
// mutated bytes, the 1-based line of the mutation, and ok=false if the
// mutator cannot apply to this source (e.g. missing-semicolon on a file
// with no `;`).
type Mutator func(src []byte) (mutated []byte, mutLine int, ok bool)

// Shape names — exported so the harness output can name the mutator that
// surfaced a violation.
const (
	ShapeUnterminatedQuote        = "unterminated-quote"
	ShapeUnterminatedBlockComment = "unterminated-block-comment"
	ShapeUnterminatedBrace        = "unterminated-brace"
	ShapeMissingSemicolon         = "missing-semicolon"
)

// Mutators returns the canonical mutator set keyed by shape. The first three
// are positive cases (the linter must produce only localized diagnostics);
// missing-semicolon is the negative case (the linter already recovers
// cleanly — a regression here would surface as outside-line diagnostics).
func Mutators() map[string]Mutator {
	return map[string]Mutator{
		ShapeUnterminatedQuote:        MutateUnterminatedQuote,
		ShapeUnterminatedBlockComment: MutateUnterminatedBlockComment,
		ShapeUnterminatedBrace:        MutateUnterminatedBrace,
		ShapeMissingSemicolon:         MutateMissingSemicolon,
	}
}

// MutateUnterminatedQuote finds the first complete `"..."` literal in src
// (outside of comments) and deletes its closing `"`. Returns ok=false when
// no complete quote is present.
func MutateUnterminatedQuote(src []byte) ([]byte, int, bool) {
	openIdx, closeIdx, found := findFirstQuote(src)
	if !found {
		return nil, 0, false
	}
	mut := make([]byte, 0, len(src)-1)
	mut = append(mut, src[:closeIdx]...)
	mut = append(mut, src[closeIdx+1:]...)
	_ = openIdx
	return mut, lineOf(src, closeIdx), true
}

// MutateUnterminatedBlockComment prepends `/* unterminated\n` to the start
// of src. The injected `/*` is on a fresh line so it doesn't nest inside an
// existing comment, and reaches EOF without a matching `*/` — the canonical
// shape from §2 of the cascade memo.
func MutateUnterminatedBlockComment(src []byte) ([]byte, int, bool) {
	if len(src) == 0 {
		return nil, 0, false
	}
	injection := []byte("/* unterminated\n")
	mut := make([]byte, 0, len(src)+len(injection))
	mut = append(mut, injection...)
	mut = append(mut, src...)
	return mut, 1, true
}

// MutateUnterminatedBrace finds the last unquoted, uncommented `}` in src
// and deletes it. Returns ok=false when no closing brace is present.
func MutateUnterminatedBrace(src []byte) ([]byte, int, bool) {
	idx, found := findLastUnquotedByte(src, '}')
	if !found {
		return nil, 0, false
	}
	mut := make([]byte, 0, len(src)-1)
	mut = append(mut, src[:idx]...)
	mut = append(mut, src[idx+1:]...)
	return mut, lineOf(src, idx), true
}

// MutateMissingSemicolon finds the first unquoted, uncommented `;` in src
// and deletes it. Negative case — the parser already recovers cleanly here
// (per §2 of the cascade memo); a regression that re-introduces a cascade
// would fail the locality property.
func MutateMissingSemicolon(src []byte) ([]byte, int, bool) {
	idx, found := findFirstUnquotedByte(src, ';')
	if !found {
		return nil, 0, false
	}
	mut := make([]byte, 0, len(src)-1)
	mut = append(mut, src[:idx]...)
	mut = append(mut, src[idx+1:]...)
	return mut, lineOf(src, idx), true
}

// Violation is a single locality breach surfaced by the harness.
type Violation struct {
	File      string
	Shape     string
	MutLine   int
	Window    int // ±W
	Diags     []scan.Diagnostic
	Lines     []DiagLine
}

// DiagLine pairs a diagnostic with the line range its span covers.
type DiagLine struct {
	Code      string
	StartLine int
	EndLine   int
	Col       int
	Message   string
}

// String renders a Violation in a format diagnostic enough to root-cause
// without re-running the M12.1 throwaway.
func (v Violation) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "locality violation: file=%s shape=%s mutLine=%d window=±%d\n",
		v.File, v.Shape, v.MutLine, v.Window)
	for _, dl := range v.Lines {
		fmt.Fprintf(&b, "  +%s at %d:%d (span lines %d..%d) %q\n",
			dl.Code, dl.StartLine, dl.Col, dl.StartLine, dl.EndLine, dl.Message)
	}
	return b.String()
}

// Check runs `mutator` against `src`, lints both clean and mutated, and
// returns a Violation pointer when any new diagnostic's [startLine,endLine]
// range fails to intersect [mutLine-window, mutLine+window]. Returns
// (nil, true, nil) on a clean pass and (nil, false, nil) when the mutator
// could not apply to this source. file is used only for diagnostic naming.
func Check(file string, src []byte, shape string, mutator Mutator, window int) (*Violation, bool, error) {
	mut, mutLine, ok := mutator(src)
	if !ok {
		return nil, false, nil
	}

	linter := lint.New()
	baselineDiags, err := linter.Lint(file, src)
	if err != nil {
		return nil, true, fmt.Errorf("baseline lint failed: %w", err)
	}
	postDiags, err := linter.Lint(file, mut)
	if err != nil {
		return nil, true, fmt.Errorf("post-mutation lint failed: %w", err)
	}

	// Index baseline by (line, col, code) so we subtract clean-file noise
	// (LINT_VE_INCONSISTENCY, the eof.block-comment-unterminated.decl
	// permanent VOID_SCAN, etc.) from the post set.
	mutLI := buildLineIndex(mut)
	baseLI := buildLineIndex(src)
	type key struct {
		line, col int
		code      string
	}
	baseSet := make(map[key]bool, len(baselineDiags))
	for _, d := range baselineDiags {
		pos := baseLI.PosAt(d.Span.Start)
		baseSet[key{pos.Line, pos.Col, d.Code}] = true
	}

	var lines []DiagLine
	var outOfWindow []scan.Diagnostic
	for _, d := range postDiags {
		start := mutLI.PosAt(d.Span.Start)
		// On a mutated source we can't meaningfully subtract baseline by
		// (line,col,code) because the mutation shifts offsets. Subtract by
		// code alone for codes that appear in both — and only treat new
		// diagnostics (codes/positions not in baseline at the SAME line)
		// as candidates. The simpler subtraction (same line on both files)
		// keeps the harness honest without trying to track offset deltas.
		if baseSet[key{start.Line, start.Col, d.Code}] {
			continue
		}
		end := mutLI.PosAt(d.Span.End)
		startLine := start.Line
		endLine := end.Line
		if endLine < startLine {
			endLine = startLine
		}
		// in-window iff [startLine, endLine] ∩ [mutLine-W, mutLine+W] ≠ ∅
		lo := mutLine - window
		hi := mutLine + window
		if endLine < lo || startLine > hi {
			outOfWindow = append(outOfWindow, scan.Diagnostic{Code: scan.DiagnosticCode(d.Code), Span: d.Span, Message: d.Message})
			lines = append(lines, DiagLine{
				Code:      string(d.Code),
				StartLine: startLine,
				EndLine:   endLine,
				Col:       start.Col,
				Message:   d.Message,
			})
		}
	}

	if len(outOfWindow) == 0 {
		return nil, true, nil
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].StartLine < lines[j].StartLine })
	return &Violation{
		File:    file,
		Shape:   shape,
		MutLine: mutLine,
		Window:  window,
		Diags:   outOfWindow,
		Lines:   lines,
	}, true, nil
}

// --- byte-walking helpers (purposely no scan.Scan dependency: the mutators
// have to work below the tokenizer so we don't accidentally exercise the
// scanner's own recovery paths in the harness setup) ---

// findFirstQuote returns (openIdx, closeIdx, true) for the first complete
// `"..."` literal in src, skipping over // and /* */ comments. Honors
// backslash-escape inside the quote.
func findFirstQuote(src []byte) (int, int, bool) {
	i := 0
	for i < len(src) {
		b := src[i]
		switch {
		case i+1 < len(src) && b == '/' && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case i+1 < len(src) && b == '/' && src[i+1] == '*':
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
		case b == '"':
			open := i
			j := i + 1
			for j < len(src) {
				if src[j] == '\\' && j+1 < len(src) {
					j += 2
					continue
				}
				if src[j] == '\n' || src[j] == '"' {
					break
				}
				j++
			}
			if j < len(src) && src[j] == '"' {
				return open, j, true
			}
			// quote that doesn't close on the same line — skip and keep looking
			i = j
			if i < len(src) {
				i++
			}
		default:
			i++
		}
	}
	return 0, 0, false
}

// findFirstUnquotedByte returns the offset of the first occurrence of `target`
// outside of quotes and comments.
func findFirstUnquotedByte(src []byte, target byte) (int, bool) {
	return scanByte(src, target, false)
}

// findLastUnquotedByte returns the offset of the last occurrence of `target`
// outside of quotes and comments.
func findLastUnquotedByte(src []byte, target byte) (int, bool) {
	return scanByte(src, target, true)
}

func scanByte(src []byte, target byte, last bool) (int, bool) {
	i := 0
	matchIdx := -1
	for i < len(src) {
		b := src[i]
		switch {
		case i+1 < len(src) && b == '/' && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case i+1 < len(src) && b == '/' && src[i+1] == '*':
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
		case b == '"':
			j := i + 1
			for j < len(src) {
				if src[j] == '\\' && j+1 < len(src) {
					j += 2
					continue
				}
				if src[j] == '"' || src[j] == '\n' {
					break
				}
				j++
			}
			if j < len(src) {
				i = j + 1
			} else {
				i = j
			}
		case b == target:
			if !last {
				return i, true
			}
			matchIdx = i
			i++
		default:
			i++
		}
	}
	if matchIdx >= 0 {
		return matchIdx, true
	}
	return 0, false
}

// lineOf returns the 1-based line number of offset within src.
func lineOf(src []byte, offset int) int {
	if offset > len(src) {
		offset = len(src)
	}
	line := 1
	for i := 0; i < offset; i++ {
		if src[i] == '\n' {
			line++
		}
	}
	return line
}

// buildLineIndex mirrors scan.BuildLineIndex without taking a dep on the
// scanner internals — the harness operates on raw bytes and reports raw
// (line, col) positions via the same logic the report layer uses.
func buildLineIndex(src []byte) scan.LineIndex {
	return scan.BuildLineIndex(src)
}
