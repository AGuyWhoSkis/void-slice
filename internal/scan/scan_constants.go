package scan

var TokenKind = struct {
	SYMBOL         Kind
	IDENTIFIER     Kind
	QUOTE_LITERAL  Kind
	NUMBER_LITERAL Kind
	COMMENT_BLOCK  Kind
	COMMENT_LINE   Kind
}{
	SYMBOL:         KindSymbol,
	IDENTIFIER:     KindIdentifier,
	QUOTE_LITERAL:  KindQuoteLiteral,
	NUMBER_LITERAL: KindNumberLiteral,
	COMMENT_BLOCK:  KindCommentBlock,
	COMMENT_LINE:   KindCommentLine,
}

const (
	KindSymbol Kind = iota
	KindIdentifier
	KindQuoteLiteral
	KindNumberLiteral
	KindCommentBlock
	KindCommentLine
)

func (tk Kind) String() string {
	switch tk {
	case KindSymbol:
		return "Sym"
	case KindIdentifier:
		return "Ident"
	case KindQuoteLiteral:
		return "Quot"
	case KindNumberLiteral:
		return "Num"
	case KindCommentBlock:
		return "Cmt-Blk"
	case KindCommentLine:
		return "Cmt-Lin"
	default:
		return "(?)"
	}
}

type DiagnosticCode string

var Codes = struct {
	SCAN           DiagnosticCode
	SCAN_STRUCTURE DiagnosticCode
}{
	SCAN:           "VOID_SCAN",
	SCAN_STRUCTURE: "VOID_SCAN_STRUCTURE",
}

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
