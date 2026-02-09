package scan

import (
	"fmt"
	"strconv"
	"strings"
)

type Source struct {
	Name  string
	Bytes []byte
	Lines *LineIndex // nil until built
}

// offsets of '\n'
// wraps []int of 0-based indexes, one per newline char (\n)
type LineIndex struct {
	Newlines []int // offsets of '\n'
}

// Describes what a token actually is
type Kind int

func NewSpan(start, end int) Span {
	return Span{
		Start: start,
		End:   end,
	}
}

type Span struct {
	Start int // byte offset, inclusive
	End   int // byte offset, exclusive
}

type Pos struct {
	Line int // 1-based
	Col  int // 1-based (in bytes)
}

type SpanPos struct {
	Span  Span
	Start Pos
	End   Pos
}

// ----- Layer 1: raw, test-friendly string forms -----

func (dc DiagnosticCode) String() string {
	return string(dc)
}

// String prints raw offsets in a compact, unambiguous form.
// I like [start,end) because it bakes in inclusive/exclusive semantics.
func (s Span) String() string {
	var b strings.Builder
	b.Grow(2 + 2*11 + 2) // rough: brackets + two ints + punctuation
	b.WriteByte('[')
	b.WriteString(strconv.Itoa(s.Start))
	b.WriteByte(',')
	b.WriteString(strconv.Itoa(s.End))
	b.WriteString(")")
	return b.String()
}

type Diagnostic struct {
	Code     DiagnosticCode // Answers "who" reported
	Severity Severity       // Answers "how bad"
	Span     Span           // Answers "where"
	Message  string         // Summary of the problem (excluding span/severity/code)
}

// Diagnostic.String is also raw-offset based by default.
// Keep it deterministic and readable; quote Message so weird chars/newlines are safe.
func (d Diagnostic) String() string {
	var b strings.Builder
	// Rough capacity guess to reduce growth; totally optional.
	b.Grow(len(d.Message) + 64)

	b.WriteString("Diagnostic{Code=")
	b.WriteString(d.Code.String()) // assume these have String() already
	b.WriteString(", Severity=")
	b.WriteString(d.Severity.String())
	b.WriteString(", Span=")
	b.WriteString(d.Span.String())
	b.WriteString(", Message=")
	b.WriteString(strconv.Quote(d.Message))
	b.WriteByte('}')

	return b.String()
}

// possible splits for Severity
// - error (guaranteed crash), warn (potential for crash), risk (potential for broken mechanics/features)
// - syntax (brackets, quotes, semicolons, etc), structure (filetype specific?), ??
// - .. (?)
type Severity int

// Represents exactly one comment, string, bracket, keyword, variable, etc.
type Token struct {
	Kind Kind
	Span Span
}

func (t Token) String() string {
	return fmt.Sprintf("%s@%s", t.Kind, t.Span)
}
