package scan

// Planned: utilities that parse/validate will likely want.
//
// Suggested utilities to implement as needed
//   [ ] Lexeme(src, tok) []byte
//   [ ] Sym(src, tok) byte
//   [ ] EqIdent(src, tok, lit string) bool
//         - Return true iff tok.Kind==IDENTIFIER and lexeme bytes equal lit
//         - Implement without allocation (compare lengths then bytes)
//   [ ] ParseIntLiteral(src, tok) (int64, ok)
//         - Only for NUMBER_LITERAL tokens (handles leading '-' + digits)
//         - Keep fast: avoid strconv if possible; or use strconv.ParseInt on string(tokBytes)
//           if acceptable (string alloc). For huge files, avoid per-token string alloc.
//         - If a fast int parser already exists, reuse it.
//
// Tests planned
//   [ ] scan_models_test.go: Lexeme/Sym/EqIdent sanity

import (
	"strconv"
	"strings"
)

func IsWhitespace(b byte) bool {
	// TODO: assess if \v \r \f are worth checking
	return b == ' ' || b == '\t' || b == '\r' || b == '\v' || b == '\f'
}

func IsDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// A number type suffix is the 'f' in 0f, or 'm' in 0m
func IsNumberTypeSuffix(b byte) bool {
	return b == 'f' || b == 'm'
}

func IsAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}

func IsIdentStart(b byte) bool {
	return IsAlpha(b) || b == '_'
}

func IsIdentCont(b byte) bool {
	return IsIdentStart(b) || IsDigit(b)
}

func IsASCIIPunct(b byte) bool {
	return (b >= '!' && b <= '/') ||
		(b >= ':' && b <= '@') ||
		(b >= '[' && b <= '`') ||
		(b >= '{' && b <= '~')
}

// HumanSpan formats Span using line:col positions, given a LineIndex
func HumanSpan(li LineIndex, s Span) string {
	start := li.PosAt(s.Start)
	end := li.PosAt(s.End)

	var b strings.Builder
	b.Grow(32)

	// Example output: "1:5-1:12"
	b.WriteString(strconv.Itoa(start.Line))
	b.WriteByte(':')
	b.WriteString(strconv.Itoa(start.Col))
	b.WriteByte('-')
	b.WriteString(strconv.Itoa(end.Line))
	b.WriteByte(':')
	b.WriteString(strconv.Itoa(end.Col))

	return b.String()
}

// If you want a human-readable diagnostic, make it explicit (no global state).
func (d Diagnostic) HumanString(li LineIndex) string {
	var b strings.Builder
	b.Grow(len(d.Message) + 80)

	b.WriteString("Diagnostic{Code=")
	b.WriteString(d.Code.String())
	b.WriteString(", Severity=")
	b.WriteString(d.Severity.String())
	b.WriteString(", Span=")
	b.WriteString(HumanSpan(li, d.Span))
	b.WriteString(", Message=")
	b.WriteString(strconv.Quote(d.Message))
	b.WriteByte('}')

	return b.String()
}

func (li LineIndex) SpanPos(sp Span) SpanPos {
	return SpanPos{
		Span:  sp,
		Start: li.PosAt(sp.Start),
		End:   li.PosAt(sp.End),
	}
}

// Creates a slice of newline byte offsets, e.g. "\nABC\n" ==> []int{3, 4}
func BuildLineIndex(src []byte) LineIndex {
	li := LineIndex{Newlines: []int{}}
	for i, b := range src {
		if b == '\n' {
			li.Newlines = append(li.Newlines, i)
		}
	}
	return li
}

func (li LineIndex) PosAt(offset int) Pos {
	// Assumes li.Newlines contains byte offsets where src[i] == '\n', in ascending order.
	prevNLOffset := -1
	line := 1

	// To compute line/col:
	// line = number of newlines before offset + 1
	// col = offset - (last newline offset) (plus 1 for 1-based col)

	for _, nl := range li.Newlines {
		if nl >= offset {
			break
		}
		prevNLOffset = nl
		line++
	}

	if prevNLOffset == -1 {
		return Pos{Line: 1, Col: offset + 1}
	}
	// col is 1-based: the byte right after '\n' is col 1
	return Pos{Line: line, Col: offset - prevNLOffset}
}
