package scan

type Source struct {
	Name  string
	Bytes []byte
	Lines *LineIndex // nil until built
}

type Diagnostic struct {
	Code     string // Indicates roughly "what" is broken
	Message  string
	Severity Severity
	Span     Span // always include a span; if unknown, use zero-length at a best guess location
	// Optional:
	// Note string
}

// offsets of '\n'
// wraps []int of 0-based indexes, one per newline char (\n)
type LineIndex struct {
	Newlines []int // offsets of '\n'
}

// possible splits for Severity
// - error (guaranteed crash), warn (potential for crash), risk (potential for broken mechanics/features)
// - syntax (brackets, quotes, semicolons, etc), structure (filetype specific?), ??
// - .. (?)
type Severity int

// Describes what a token actually is
type Kind int

// Represents exactly one comment, string, bracket, keyword, variable, etc.
type Token struct {
	Kind Kind
	Span Span
}

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
