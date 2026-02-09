package scan

type ScanState int

const (
	// Scan state (multi-byte tokens)
	SCANNING_BLOCK_COMMENT ScanState = iota
	SCANNING_LINE_COMMENT
	SCANNING_QUOTE_LITERAL
	SCANNING_NUMBER_LITERAL
	SCANNING_IDENTIFIER
	SCANNING_STANDARD // first skips whitespace, then checks for entry into other scanning states, then checks for single-byte tokens like {}()[],.-=+; etc.
	SCANNING_PANIC    = -1
)

// Token kinds needed for Stage 2:
// 	- punctuation: { } [ ] ( ) = ; , (maybe . depending)
// 	- identifiers: edit, m_rules, enumItem, item, num, m_name, etc
// 	- number literals: 34, -1, 23.89 (float maybe)
// 	- string literals: "..." (with escapes)
// 	- comment tokens: line and block (or simply skip them)
// 	- newlines/whitespace usually skipped

var TokenKind = struct {
	SYMBOL         Kind
	IDENTIFIER     Kind
	QUOTE_LITERAL  Kind
	NUMBER_LITERAL Kind
	COMMENT_BLOCK  Kind
	COMMENT_LINE   Kind
	NEWLINE        Kind
	WHITESPACE     Kind
}{
	SYMBOL:         kindSymbol,
	IDENTIFIER:     kindIdentifier,
	QUOTE_LITERAL:  kindQuoteLiteral,
	NUMBER_LITERAL: kindNumberLiteral,
	COMMENT_BLOCK:  kindCommentBlock,
	COMMENT_LINE:   kindCommentLine,
	NEWLINE:        kindNewline,
	WHITESPACE:     kindWhitespace,
}

const (
	kindSymbol Kind = iota
	kindIdentifier
	kindQuoteLiteral
	kindNumberLiteral
	kindCommentBlock
	kindCommentLine
	kindNewline
	kindWhitespace
)

func (tk Kind) String() string {
	switch tk {
	case kindSymbol:
		return "symbol"
	case kindIdentifier:
		return "identifier"
	case kindQuoteLiteral:
		return "dbl-quote-literal"
	case kindNumberLiteral:
		return "number-literal"
	case kindCommentBlock:
		return "comment-block"
	case kindCommentLine:
		return "comment-line"
	case kindNewline:
		return "new-line"
	case kindWhitespace:
		return "whitespace"
	default:
		return "UNDEFINED"
	}
}

type DiagnosticCode string

var Codes = struct {
	SCAN           DiagnosticCode
	SCAN_STRUCTURE DiagnosticCode
}{
	SCAN:           scan,
	SCAN_STRUCTURE: codeStructure,
}

const (
	scan          DiagnosticCode = "VOID_SCAN"
	codeStructure DiagnosticCode = "VOID_SCAN_STRUCTURE"
)

// 	Lexical diagnostics:
// 		- unterminated string
// 		- unterminated block comment
// 		- invalid escape sequence (if you enforce)
//	 	- invalid byte / unexpected control chars (optional)

const (
	NIL   Severity = iota // (0) nil value
	RISK                  // denotes 'this risks breaking something', or 'this edge case that you might not care about crashes'
	WARN                  // denotes 'this might cause a crash', or 'sometimes, this will crash'
	ERROR                 // denotes 'this will cause a crash'
	PANIC                 // denotes 'something went wrong while scanning this'
)

var Severities = struct {
	NIL   Severity
	RISK  Severity
	WARN  Severity
	ERROR Severity
	PANIC Severity
}{
	NIL:   NIL,
	RISK:  RISK,
	WARN:  WARN,
	ERROR: ERROR,
	PANIC: PANIC,
}

func (s Severity) String() string {
	switch s {
	case NIL:
		return "NIL"
	case RISK:
		return "RISK"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case PANIC:
		return "PANIC"
	default:
		return "DNE" // would panic() instead here be helpful?
	}
}
