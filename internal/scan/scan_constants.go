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
}{
	SYMBOL:         kindSymbol,
	IDENTIFIER:     kindIdentifier,
	QUOTE_LITERAL:  kindQuoteLiteral,
	NUMBER_LITERAL: kindNumberLiteral,
	COMMENT_BLOCK:  kindCommentBlock,
	COMMENT_LINE:   kindCommentLine,
	NEWLINE:        kindNewline,
}

const (
	kindSymbol Kind = iota
	kindIdentifier
	kindQuoteLiteral
	kindNumberLiteral
	kindCommentBlock
	kindCommentLine
	kindNewline
)

// 	Lexical diagnostics:
// 		- unterminated string
// 		- unterminated block comment
// 		- invalid escape sequence (if you enforce)
//	 	- invalid byte / unexpected control chars (optional)
