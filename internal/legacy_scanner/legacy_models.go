package scanner

import (
	"void-slice/internal/scan"
)

type LexState int

type Severity int

// Describes what a token actually is?

type TokenKind int
const (
	// Scan state
	SCANNING_BLOCK_COMMENT LexState = iota
	SCANNING_LINE_COMMENT
	SCANNING_DOUBLE_QUOTE
	SCANNING_STANDARD
	SCANNING_PANIC = -1

	// Severity codes
	RISK  Severity = iota // denotes 'this risks breaking something', or 'this edge case that you might not care about crashes'
	WARN                  // denotes 'this might cause a crash', or 'sometimes, this will crash'
	ERROR                 // denotes 'this will cause a crash'

	// TokenKind                                    examples
	DECL_NIL                 TokenKind = iota // nil value
	DECL_BLOCK_COMMENT                        /* block comments */
	DECL_LINE_COMMENT                         // 1-line comments with double slash //
	DECL_DOUBLE_QUOTE                         // "quoted strings" including "escaped \"quotes like so" (valid str)
	DECL_BLOCK_LABEL                          // edit (ie. from edit = {..}) // TODO: are these needed?
	DECL_FIELD_ARRAY_KEY                      // arr[ATTLV_SEARCH]
	DECL_FIELD_ARRAY_VAL                      // 0,
	DECL_FIELD_VAL_KEY                        // m_defaultMeterToRaise = ..
	DECL_SYMBOL_TERMINATE                     // ;
	DECL_SYMBOL_FIELD_ASSIGN                  // m_x = y;
	DECL_SYMBOL_BLOCK_ASSIGN                  // edit = { .. }
	DECL_NUMBER                               // int or float
	DECL_IDENTIFIER
	DECL_BRACE_SQUIG_OPEN
	DECL_BRACE_SQUIG_CLOSE
	DECL_BRACE_ROUND_OPEN
	DECL_BRACE_ROUND_CLOSE
	DECL_BRACE_SQUARE_OPEN
	DECL_BRACE_SQUARE_CLOSE
)

const (
	VoidSyntax    string = "VOID-SYNTAX"    // linting errors, like missing semicolon ; or missing brackets {}[]()
	VoidStartup   string = "VOID-STARTUP"   // errors that usually happen before any mission load
	VoidRuntime   string = "VOID-RUNTIME"   // errors that usually happen after or during mission load
	VoidStructure string = "VOID-STRUCTURE" // ie. editing checksums, or editing string of item_add["aFD3WZ"] = ..
	VoidBlunder   string = "VOID-BLUNDER"   // ie. user accidentally hung their Knight
	// ..
)

func IsWhitespace(b byte) bool {
	// TODO: assess if \v \r \f are worth checking
	return b == ' ' || b == '\t' || b == '\r' || b == '\v' || b == '\f'
}

func IsDigit(b byte) bool {
	return b >= '0' && b <= '9'
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

func IsSymbol(b byte) bool {
	return b == '=' || b == '-' || b == '+'
}

type Diagnostic struct {
	Code     string // Indicates roughly "what" is broken
	Message  string
	Severity Severity
	Span     scan.Span // always include a span; if unknown, use zero-length at a best guess location
	// Optional:
	// Note string
}

// Represents exactly one comment, string, bracket, keyword, variable, etc.
type Token struct {
	Kind TokenKind
	Span scan.Span
}
