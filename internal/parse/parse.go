package parse

import (
	"strconv"
	"void-slice/internal/scan"
)

// IMPORTANT DESIGN CONSTRAINTS:
//   - Option A punctuation: keep tokens.Kind==SYMBOL and inspect src[t.Span.Start].
//   - Huge files: parsing should be streaming-ish; do not build a full-file AST.
//     Parse one component at a time; allow validator to process and discard.

// -------------------------
// 1) Public API
// -------------------------

// Handler receives structural events as the parser walks an entities file.
// Note: validate.go will implement Handler to track object frames and run checks at ObjectEnd.
type Handler interface {
	OnVersion(versionTok scan.Token, versionValue int64)
	OnComponentBegin(componentTok scan.Token, lbrace scan.Token)
	OnComponentDecl(typeTok, nameTok scan.Token, lbrace scan.Token) // e.g. cpntFoo name {
	OnComponentEnd(rbrace scan.Token)

	OnObjectBegin(lbrace scan.Token) // for nested { ... }
	OnObjectEnd(rbrace scan.Token)

	OnAssignment(key Key, eqTok scan.Token, value Value, semiTok scan.Token)
	OnTypedBlock(typeTok scan.Token, nameTok scan.Token, lbrace scan.Token) // nameTok is zero for bare ident { ... }

	OnDiag(diag scan.Diagnostic)
}

// WalkEntities walks an entities file emitting events to h.
// Both the returned diags slice and h.OnDiag carry parse diagnostics; they are identical.
func WalkEntities(src []byte, toks []scan.Token, h Handler) (diags []scan.Diagnostic) {
	c := &cursor{
		src:   src,
		toks:  toks,
		i:     -1,
		nToks: len(toks),
	}
	c.walkEntities(h)
	return c.diags
}

// -------------------------
// 2) Cursor / token stream helper
// -------------------------

type cursor struct {
	src   []byte
	toks  []scan.Token
	i     int // -1 = before first token; i = index of last consumed token
	nToks int
	diags []scan.Diagnostic

	// eofCascadeEmitted tracks whether an EOF-anchored "expected symbol"
	// diagnostic has already fired for this walk. Inner→outer expectSym calls
	// at EOF (e.g. closing an inner object then the outer top-level) describe
	// the same missing close — the second emission is a cascade. M12.5.
	eofCascadeEmitted bool
}

// eof returns true when there are no more tokens to consume.
func (c *cursor) eof() bool {
	return c.i+1 >= c.nToks
}

// peek returns the next token without consuming it, or nil at EOF.
func (c *cursor) peek() *scan.Token {
	if c.eof() {
		return nil
	}
	return &c.toks[c.i+1]
}

// next consumes and returns the next token, or nil at EOF.
func (c *cursor) next() *scan.Token {
	if c.eof() {
		return nil
	}
	c.i++
	return &c.toks[c.i]
}

// lexeme returns the source bytes for tok.
func (c *cursor) lexeme(tok scan.Token) []byte {
	return c.src[tok.Span.Start:tok.Span.End]
}

// isIdent returns true iff tok is an IDENTIFIER whose text equals lit, without allocating.
func (c *cursor) isIdent(tok scan.Token, lit string) bool {
	if tok.Kind != scan.KindIdentifier {
		return false
	}
	if tok.Span.End-tok.Span.Start != len(lit) {
		return false
	}
	for i := 0; i < len(lit); i++ {
		if c.src[tok.Span.Start+i] != lit[i] {
			return false
		}
	}
	return true
}

// diagSpan returns a zero-length span anchored where a missing token should
// have appeared. When the parser is at EOF, anchor at len(src) so the
// diagnostic lands on the line the missing closer would have occupied (M12.5,
// per kanban/goals/M12-cascade-investigation.md §2 row 3). Otherwise anchor
// just past the last consumed token.
func (c *cursor) diagSpan() scan.Span {
	if c.eof() {
		return c.eofSpan()
	}
	if c.i >= 0 && c.i < c.nToks {
		end := c.toks[c.i].Span.End
		return scan.NewSpan(end, end)
	}
	return scan.NewSpan(0, 0)
}

// eofSpan returns a zero-length span at the position a missing closing token
// would occupy. Mapped to (line, col) by the renderer, this lands on the
// line the user would edit.
//
// Two anchor cases (M12.6, expanding M12.5):
//
//   - When src ends with two or more newlines (LF `\n\n` or CRLF
//     `\r\n\r\n`), anchor right after the first newline of the trailing
//     pair. This puts the diagnostic on the empty line between the two
//     newlines — the line the deleted `}` was originally on for files
//     whose `}\n`-tail was the source of the mutation. PosAt mapping of
//     `len(src)` would land one line further down, which is past the
//     user-edit line.
//   - Otherwise, anchor at `len(src)`. For `}` (no trailing newline) this
//     lands on the last content line; for `}\n` (single trailing newline,
//     the committed cascade fixture shape) it lands on the empty line
//     just past content. Both are correct.
func (c *cursor) eofSpan() scan.Span {
	if p := firstTrailingNewlinePairOffset(c.src); p >= 0 {
		anchor := p + 1
		return scan.NewSpan(anchor, anchor)
	}
	return scan.NewSpan(len(c.src), len(c.src))
}

// lastConsumedEndSpan returns a zero-length span at the end of the most
// recently consumed token, or (0,0) if no tokens have been consumed. Used
// for diagnostics whose user-edit position is "right after the last token I
// just read" (e.g. a missing `;` after an assignment value). Distinct from
// eofSpan, which targets the line a missing block-closer would occupy — for
// statement terminators at EOF, that policy would land one line past the
// value token, which is not where the user types the `;`.
func (c *cursor) lastConsumedEndSpan() scan.Span {
	if c.i >= 0 && c.i < c.nToks {
		end := c.toks[c.i].Span.End
		return scan.NewSpan(end, end)
	}
	return scan.NewSpan(0, 0)
}

// firstTrailingNewlinePairOffset returns the offset of the first `\n` of a
// trailing newline pair in src — i.e. src ends in `\n\n` (LF) or
// `\r\n\r\n` (CRLF). Returns -1 otherwise. CRLF is handled by stripping a
// `\r` immediately before either `\n`; the returned offset is always that
// of a `\n` byte, never a `\r`.
func firstTrailingNewlinePairOffset(src []byte) int {
	n := len(src)
	if n < 2 || src[n-1] != '\n' {
		return -1
	}
	// Skip the trailing newline (and an optional `\r` for CRLF).
	end := n - 1
	if end > 0 && src[end-1] == '\r' {
		end--
	}
	if end == 0 {
		return -1
	}
	if src[end-1] != '\n' {
		return -1
	}
	return end - 1
}

// scannerAteEOF returns true when the scanner's last token is an unterminated
// block comment that consumed bytes to EOF. The parser uses this to suppress
// "expected '}'" cascade diagnostics for structural tokens lost inside the
// over-extended comment — the VOID_SCAN already pinpoints the fault (M12.4).
//
// The check is local: an unterminated block comment is a COMMENT_BLOCK whose
// Span.End hits len(src) but whose tail bytes aren't `*/`. No scanDiags
// plumbing required.
func (c *cursor) scannerAteEOF() bool {
	if len(c.toks) == 0 {
		return false
	}
	last := c.toks[len(c.toks)-1]
	if last.Span.End != len(c.src) || last.Kind != scan.KindCommentBlock {
		return false
	}
	n := last.Span.End - last.Span.Start
	return !(n >= 4 && c.src[last.Span.End-2] == '*' && c.src[last.Span.End-1] == '/')
}

// emitDiag records a diagnostic in c.diags and forwards it to h.
func (c *cursor) emitDiag(h Handler, d scan.Diagnostic) {
	c.diags = append(c.diags, d)
	h.OnDiag(d)
}

// matchKind peeks and, if the next token has kind k, consumes and returns it.
func (c *cursor) matchKind(k scan.Kind) (scan.Token, bool) {
	tok := c.peek()
	if tok == nil || tok.Kind != k {
		return scan.Token{}, false
	}
	return *c.next(), true
}

// expectKind is like matchKind but emits a diagnostic if the match fails.
func (c *cursor) expectKind(k scan.Kind, code scan.DiagnosticCode, msg string) (scan.Token, bool) {
	tok, ok := c.matchKind(k)
	if !ok {
		if c.suppressEOFDiag() {
			return tok, ok
		}
		span := c.diagSpan()
		c.diags = append(c.diags, scan.Diagnostic{Code: code, Span: span, Message: msg})
		if c.eof() {
			c.eofCascadeEmitted = true
		}
	}
	return tok, ok
}

// matchSym peeks and, if the next token is a SYMBOL matching ch, consumes and returns it.
func (c *cursor) matchSym(ch byte) (scan.Token, bool) {
	tok := c.peek()
	if tok == nil || tok.Kind != scan.KindSymbol || c.src[tok.Span.Start] != ch {
		return scan.Token{}, false
	}
	return *c.next(), true
}

// expectSym is like matchSym but emits a diagnostic if the match fails.
func (c *cursor) expectSym(ch byte, code scan.DiagnosticCode, msg string) (scan.Token, bool) {
	tok, ok := c.matchSym(ch)
	if !ok {
		if c.suppressEOFDiag() {
			return tok, ok
		}
		span := c.diagSpan()
		c.diags = append(c.diags, scan.Diagnostic{Code: code, Span: span, Message: msg})
		if c.eof() {
			c.eofCascadeEmitted = true
		}
	}
	return tok, ok
}

// suppressEOFDiag returns true if the parser is at EOF and should skip
// emitting another "expected X" diagnostic. Two cases:
//   - scannerAteEOF: the scanner already emitted VOID_SCAN for an unterminated
//     block comment that swallowed structural tokens (M12.4).
//   - eofCascadeEmitted: an outer expect already emitted at EOF for the same
//     missing closer (M12.5 dedupe).
func (c *cursor) suppressEOFDiag() bool {
	if !c.eof() {
		return false
	}
	return c.scannerAteEOF() || c.eofCascadeEmitted
}

// tokenIsUnterminatedQuote reports whether tok is a quote literal the scanner
// emitted as part of a VOID_SCAN unterminated-quote fault — its lexeme starts
// with `"` but doesn't end with a closing `"`. Used to suppress follow-up
// parse cascade diagnostics (missing semicolon) on a statement whose value
// was already reported scan-faulted (M12.3).
func (c *cursor) tokenIsUnterminatedQuote(t scan.Token) bool {
	if t.Kind != scan.KindQuoteLiteral {
		return false
	}
	n := t.Span.End - t.Span.Start
	if n < 2 {
		return true
	}
	return c.src[t.Span.End-1] != '"'
}

// matchIdent peeks and, if the next token is an IDENTIFIER matching lit, consumes and returns it.
func (c *cursor) matchIdent(lit string) (scan.Token, bool) {
	tok := c.peek()
	if tok == nil || !c.isIdent(*tok, lit) {
		return scan.Token{}, false
	}
	return *c.next(), true
}

// syncTo advances until the peeked token is a SYMBOL matching sym1 or (if sym2!=0) sym2.
// It stops WITHOUT consuming the matching token so the caller can consume it explicitly.
func (c *cursor) syncTo(sym1 byte, sym2 byte) {
	for {
		tok := c.peek()
		if tok == nil {
			return
		}
		if tok.Kind == scan.KindSymbol {
			b := c.src[tok.Span.Start]
			if b == sym1 || (sym2 != 0 && b == sym2) {
				return
			}
		}
		c.next()
	}
}

// -------------------------
// 3) Minimal parse data types
// -------------------------

type Key struct {
	BaseTok  scan.Token
	Indexers []Indexer // zero or more
}

type IndexerKind int

const (
	IndexInt IndexerKind = iota
	IndexString
	IndexIdent
)

type Indexer struct {
	LBrackTok scan.Token
	ValueTok  scan.Token // NUMBER_LITERAL or QUOTE_LITERAL
	RBrackTok scan.Token
	Kind      IndexerKind
	IntValue  int64 // set when Kind==IndexInt
}

type ValueKind int

const (
	ValNumber ValueKind = iota
	ValString
	ValIdent
	ValObject
	ValTuple // ( v1, v2, ... ) — opaque payload, contents not inspected
)

type Value struct {
	Kind ValueKind
	Tok  scan.Token // for number/string/ident
	Obj  ObjectSpan // for object value
	Tup  TupleSpan  // for tuple value
}

type ObjectSpan struct {
	LBraceTok scan.Token
	RBraceTok scan.Token
}

type TupleSpan struct {
	LParenTok scan.Token
	RParenTok scan.Token
}

// -------------------------
// 4) Grammar / walk functions
// -------------------------

func (c *cursor) walkEntities(h Handler) {
	// Shape-1 .decl bare-curly form: file is just `{ ...statements... }`.
	// Skip leading comments to find the first significant token.
	for {
		p := c.peek()
		if p == nil {
			return
		}
		if p.Kind == scan.KindCommentLine || p.Kind == scan.KindCommentBlock {
			c.next()
			continue
		}
		break
	}
	if p := c.peek(); p != nil && p.Kind == scan.KindSymbol && c.src[p.Span.Start] == '{' {
		lbrace := c.next()
		h.OnObjectBegin(*lbrace)
		c.walkObjectBody(h)
		rbrace, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close top-level object")
		if !ok {
			c.syncTo('}', 0)
			rbrace, _ = c.matchSym('}')
		}
		h.OnObjectEnd(rbrace)
		return
	}

	// Version <number>
	versionTok, ok := c.matchIdent("Version")
	if ok {
		numTok, numOk := c.expectKind(scan.KindNumberLiteral, Codes.UNEXPECTED_TOKEN, "expected version number after 'Version'")
		if numOk {
			val, _ := parseIntLiteral(c.lexeme(numTok))
			h.OnVersion(versionTok, val)
		}
	}

	for !c.eof() {
		tok := c.peek()
		if tok == nil {
			break
		}
		if tok.Kind == scan.KindIdentifier && c.isIdent(*tok, "component") {
			c.walkComponent(h)
		} else if tok.Kind == scan.KindIdentifier {
			// Unknown top-level identifier — consume it, then check what follows.
			unknownTok := c.next()
			p := c.peek()
			if p != nil && p.Kind == scan.KindSymbol && c.src[p.Span.Start] == '{' {
				// Unknown block (e.g. "entity { ... }") — walk body silently.
				lbrace := c.next()
				h.OnTypedBlock(*unknownTok, scan.Token{}, *lbrace)
				c.walkObjectBody(h)
				if _, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close unknown top-level block"); !ok {
					c.syncTo('}', 0)
					c.matchSym('}')
				}
			} else {
				// Not a block — emit one diagnostic and continue.
				c.emitDiag(h, scan.Diagnostic{
					Code:    Codes.UNEXPECTED_TOKEN,
					Span:    unknownTok.Span,
					Message: "expected 'component'",
				})
			}
		} else {
			c.emitDiag(h, scan.Diagnostic{
				Code:    Codes.UNEXPECTED_TOKEN,
				Span:    tok.Span,
				Message: "unexpected token at top level",
			})
			c.next()
		}
	}
}

func (c *cursor) walkComponent(h Handler) {
	componentTok := c.next() // consume "component" (already peeked by caller)

	lbrace, ok := c.expectSym('{', Codes.EXPECTED_SYMBOL, "expected '{' after 'component'")
	if !ok {
		c.syncTo('}', 0)
		c.matchSym('}')
		return
	}
	h.OnComponentBegin(*componentTok, lbrace)

	typeTok, typeOk := c.expectKind(scan.KindIdentifier, Codes.EXPECTED_IDENTIFIER, "expected type identifier in component declaration")
	nameTok, nameOk := c.expectKind(scan.KindIdentifier, Codes.EXPECTED_IDENTIFIER, "expected name identifier in component declaration")
	declLBrace, declOk := c.expectSym('{', Codes.EXPECTED_SYMBOL, "expected '{' to open component body")

	if !typeOk || !nameOk || !declOk {
		// recover: skip to end of component
		c.syncTo('}', 0)
		c.matchSym('}')
		rbrace, _ := c.matchSym('}')
		h.OnComponentEnd(rbrace)
		return
	}

	h.OnComponentDecl(typeTok, nameTok, declLBrace)
	c.walkObjectBody(h)

	// close inner decl body
	if _, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close component body"); !ok {
		c.syncTo('}', 0)
		c.matchSym('}')
	}

	// close outer component block
	rbrace, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close component block")
	if !ok {
		c.syncTo('}', 0)
		rbrace, _ = c.matchSym('}')
	}
	h.OnComponentEnd(rbrace)
}

// walkObjectBody parses statements until it sees '}' or EOF.
// It does NOT consume the closing '}'.
func (c *cursor) walkObjectBody(h Handler) {
	for {
		tok := c.peek()
		if tok == nil {
			// EOF while inside object — unterminated
			return
		}
		if tok.Kind == scan.KindSymbol && c.src[tok.Span.Start] == '}' {
			return
		}
		switch tok.Kind {
		case scan.KindIdentifier:
			c.walkStatement(h)
		case scan.KindCommentLine, scan.KindCommentBlock, scan.KindQuoteLiteral:
			// Skip comments and quoted type annotations (e.g. "ns::Type" before an assignment).
			c.next()
		default:
			c.emitDiag(h, scan.Diagnostic{
				Code:    Codes.UNEXPECTED_TOKEN,
				Span:    tok.Span,
				Message: "unexpected token in object body",
			})
			c.next() // always advance to prevent infinite loop
		}
	}
}

// walkStatement parses one statement starting with an IDENTIFIER.
func (c *cursor) walkStatement(h Handler) {
	baseTok := c.next() // consume base IDENT

	// Parse optional indexers: [N] or ["str"]
	var indexers []Indexer
	for {
		lb, ok := c.matchSym('[')
		if !ok {
			break
		}
		valTok := c.peek()
		var idx Indexer
		idx.LBrackTok = lb
		if valTok == nil {
			c.diags = append(c.diags, scan.Diagnostic{
				Code:    Codes.UNEXPECTED_TOKEN,
				Span:    c.diagSpan(),
				Message: "expected index value inside '['",
			})
			break
		}
		if valTok.Kind == scan.KindNumberLiteral {
			idx.Kind = IndexInt
			idx.ValueTok = *c.next()
			n, _ := parseIntLiteral(c.lexeme(idx.ValueTok))
			idx.IntValue = n
		} else if valTok.Kind == scan.KindQuoteLiteral {
			idx.Kind = IndexString
			idx.ValueTok = *c.next()
		} else if valTok.Kind == scan.KindIdentifier {
			idx.Kind = IndexIdent
			idx.ValueTok = *c.next()
		} else {
			c.emitDiag(h, scan.Diagnostic{
				Code:    Codes.UNEXPECTED_TOKEN,
				Span:    valTok.Span,
				Message: "expected number, string, or identifier index inside '['",
			})
			c.syncTo(']', '}')
		}
		rb, rbOk := c.expectSym(']', Codes.EXPECTED_SYMBOL, "expected ']' to close indexer")
		if rbOk {
			idx.RBrackTok = rb
		}
		indexers = append(indexers, idx)
	}

	key := Key{BaseTok: *baseTok, Indexers: indexers}

	next := c.peek()
	if next == nil {
		if !c.suppressEOFDiag() {
			c.diags = append(c.diags, scan.Diagnostic{
				Code:    Codes.UNEXPECTED_TOKEN,
				Span:    c.diagSpan(),
				Message: "unexpected end of file in statement",
			})
			c.eofCascadeEmitted = true
		}
		return
	}

	// Assignment: key = value [;]
	if next.Kind == scan.KindSymbol && c.src[next.Span.Start] == '=' {
		eqTok := c.next()
		val := c.parseValue(h)

		// M12.8: assignment-as-block typo. If parseValue returned a scalar
		// IDENT and the next token is `=`, the IDENT was actually the next
		// statement's key — the opening `{` after the original `=` was
		// forgotten. Re-anchor a focused diagnostic on the `=`, rewind the
		// misread IDENT, and treat `eqTok` as a virtual `{` so the inner
		// statements parse normally and the user's intended `}` closes the
		// synthetic object. Mirrors M12.7's soft-recovery shape (one call
		// site, no contract change to parseValue).
		if val.Kind == ValIdent {
			if p := c.peek(); p != nil && p.Kind == scan.KindSymbol && c.src[p.Span.Start] == '=' {
				c.emitDiag(h, scan.Diagnostic{
					Code:    Codes.EXPECTED_SYMBOL,
					Span:    eqTok.Span,
					Message: "expected '{' after '='",
				})
				c.i-- // un-consume the misread IDENT
				h.OnObjectBegin(*eqTok)
				c.walkObjectBody(h)
				rbrace, _ := c.closeObjectValue(h, *eqTok)
				h.OnObjectEnd(rbrace)
				objVal := Value{Kind: ValObject, Obj: ObjectSpan{LBraceTok: *eqTok, RBraceTok: rbrace}}
				h.OnAssignment(key, *eqTok, objVal, scan.Token{})
				return
			}
		}

		var semiTok scan.Token
		if val.Kind != ValObject {
			semi, ok := c.matchSym(';')
			if !ok {
				if !(val.Kind == ValString && c.tokenIsUnterminatedQuote(val.Tok)) {
					// Anchor right after the value token, not at the EOF
					// block-close line — `;` belongs immediately after the
					// expression the user just wrote, even at EOF (M12.6).
					c.emitDiag(h, scan.Diagnostic{
						Code:    Codes.EXPECTED_SEMICOLON,
						Span:    c.lastConsumedEndSpan(),
						Message: "expected ';' after assignment",
					})
				}
				// Don't sync: value was already consumed; let walkObjectBody
				// handle whatever comes next as a new statement.
			}
			semiTok = semi
		}
		h.OnAssignment(key, *eqTok, val, semiTok)
		return
	}

	// Typed block (two-ident form): TypeIdent NameIdent { body }
	if len(indexers) == 0 && next.Kind == scan.KindIdentifier {
		nameTok := c.next()
		lbrace, ok := c.expectSym('{', Codes.EXPECTED_SYMBOL, "expected '{' after typed block header")
		if !ok {
			c.syncTo('}', 0)
			return
		}
		h.OnTypedBlock(*baseTok, *nameTok, lbrace)
		c.walkObjectBody(h)
		if _, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close typed block"); !ok {
			c.syncTo('}', 0)
			c.matchSym('}')
		}
		return
	}

	// Bare block (one-ident form): ident { body }
	if len(indexers) == 0 && next.Kind == scan.KindSymbol && c.src[next.Span.Start] == '{' {
		lbrace := c.next()
		h.OnTypedBlock(*baseTok, scan.Token{}, *lbrace)
		c.walkObjectBody(h)
		if _, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close bare block"); !ok {
			c.syncTo('}', 0)
			c.matchSym('}')
		}
		return
	}

	c.emitDiag(h, scan.Diagnostic{
		Code:    Codes.UNEXPECTED_TOKEN,
		Span:    next.Span,
		Message: "unexpected token after identifier",
	})
	c.syncTo(';', '}')
}

// parseValue parses a value expression (object, string, number, or ident).
func (c *cursor) parseValue(h Handler) Value {
	tok := c.peek()
	if tok == nil {
		if !c.suppressEOFDiag() {
			c.diags = append(c.diags, scan.Diagnostic{
				Code:    Codes.UNEXPECTED_TOKEN,
				Span:    c.diagSpan(),
				Message: "expected value",
			})
			c.eofCascadeEmitted = true
		}
		return Value{}
	}

	// Object value: { ... }
	if tok.Kind == scan.KindSymbol && c.src[tok.Span.Start] == '{' {
		lbrace := c.next()
		h.OnObjectBegin(*lbrace)
		c.walkObjectBody(h)
		rbrace, _ := c.closeObjectValue(h, *lbrace)
		h.OnObjectEnd(rbrace)
		return Value{Kind: ValObject, Obj: ObjectSpan{LBraceTok: *lbrace, RBraceTok: rbrace}}
	}

	// Tuple value: ( v1, v2, ... )
	// Used by Shape-1 grammar: `color = ( 1, 1, 1, 1 );`. Contents are
	// consumed opaquely — no events, no inner validation.
	if tok.Kind == scan.KindSymbol && c.src[tok.Span.Start] == '(' {
		lparen := c.next()
		for {
			t := c.peek()
			if t == nil {
				c.emitDiag(h, scan.Diagnostic{
					Code:    Codes.UNEXPECTED_TOKEN,
					Span:    c.diagSpan(),
					Message: "unterminated tuple value",
				})
				return Value{Kind: ValTuple, Tup: TupleSpan{LParenTok: *lparen}}
			}
			if t.Kind == scan.KindSymbol && c.src[t.Span.Start] == ')' {
				rparen := c.next()
				return Value{Kind: ValTuple, Tup: TupleSpan{LParenTok: *lparen, RParenTok: *rparen}}
			}
			c.next()
		}
	}

	if tok.Kind == scan.KindQuoteLiteral {
		return Value{Kind: ValString, Tok: *c.next()}
	}
	if tok.Kind == scan.KindNumberLiteral {
		return Value{Kind: ValNumber, Tok: *c.next()}
	}
	if tok.Kind == scan.KindIdentifier {
		return Value{Kind: ValIdent, Tok: *c.next()}
	}

	c.emitDiag(h, scan.Diagnostic{
		Code:    Codes.UNEXPECTED_TOKEN,
		Span:    tok.Span,
		Message: "expected value (string, number, identifier, or object)",
	})
	c.syncTo(';', '}')
	return Value{}
}

// closeObjectValue closes the object opened by lbrace. M12.7: if the next
// `}` is at strictly lower line-indent than lbrace, treat it as belonging
// to an outer scope — emit PARSE_UNTERMINATED_OBJECT anchored at lbrace and
// leave the `}` unconsumed for the outer level to pick up. This re-anchors
// the mid-file missing-brace cascade onto the actual fault site instead of
// EOF. On any other failure shape we fall back to expectSym +
// UNTERMINATED_OBJECT + sync recovery — the M12.5 EOF-cascade behaviour.
func (c *cursor) closeObjectValue(h Handler, lbrace scan.Token) (scan.Token, bool) {
	if c.peekBraceBelongsToOuter(lbrace) {
		c.emitDiag(h, scan.Diagnostic{
			Code:    Codes.UNTERMINATED_OBJECT,
			Span:    scan.NewSpan(lbrace.Span.Start, lbrace.Span.End),
			Message: "unterminated object",
		})
		// outer expectSym('}') for the same missing close is a cascade —
		// suppress its EOF re-emission (mirrors M12.5's eofCascadeEmitted
		// dedupe).
		c.eofCascadeEmitted = true
		return scan.Token{}, false
	}
	rbrace, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close object value")
	if !ok {
		c.emitDiag(h, scan.Diagnostic{
			Code:    Codes.UNTERMINATED_OBJECT,
			Span:    scan.NewSpan(lbrace.Span.Start, c.diagSpan().Start),
			Message: "unterminated object",
		})
		c.syncTo('}', 0)
		rbrace, _ = c.matchSym('}')
	}
	return rbrace, ok
}

// peekBraceBelongsToOuter returns true iff the next token is `}` AND its
// line-indent is strictly less than lbrace's line-indent. Soft signal — see
// M12.7 ticket for the brittleness vs coverage trade-off.
func (c *cursor) peekBraceBelongsToOuter(lbrace scan.Token) bool {
	p := c.peek()
	if p == nil || p.Kind != scan.KindSymbol || c.src[p.Span.Start] != '}' {
		return false
	}
	return c.lineIndent(p.Span.Start) < c.lineIndent(lbrace.Span.Start)
}

// lineIndent returns the 1-based byte column of the first non-whitespace
// byte on the line containing offset. Spaces and tabs count one column each
// (matches the rest of the linter's column model — see scan.LineIndex).
func (c *cursor) lineIndent(offset int) int {
	start := offset
	for start > 0 && c.src[start-1] != '\n' {
		start--
	}
	i := start
	for i < len(c.src) && (c.src[i] == ' ' || c.src[i] == '\t') {
		i++
	}
	return i - start + 1
}

// parseIntLiteral parses a NUMBER_LITERAL lexeme into an int64.
// Strips trailing type suffixes (f, m) and ignores any decimal part.
func parseIntLiteral(b []byte) (int64, bool) {
	s := string(b)
	// strip suffix
	if len(s) > 0 && (s[len(s)-1] == 'f' || s[len(s)-1] == 'm') {
		s = s[:len(s)-1]
	}
	// strip decimal
	for i, ch := range s {
		if ch == '.' {
			s = s[:i]
			break
		}
	}
	if s == "" || s == "-" {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
