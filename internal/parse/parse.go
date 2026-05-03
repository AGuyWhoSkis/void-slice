package parse

import (
	"fmt"
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

// Opts configures behavior of WalkEntities. The zero value imposes no limits
// — appropriate for CLI / LSP / native callers that run on end users' machines.
// The Worker entry point (cmd/voidslice-wasm) sets MaxDiagnostics > 0 to bound
// memory usage inside its 128 MB isolate.
type Opts struct {
	// MaxDiagnostics caps the total number of diagnostics emitted by a single
	// WalkEntities call. 0 = unlimited. When the cap is reached, parsing
	// continues internally (cheap) but no more diagnostics flow to either the
	// returned slice or the Handler; the final entry becomes a sentinel with
	// code Codes.DIAGNOSTICS_TRUNCATED.
	MaxDiagnostics int
}

// WalkEntities walks an entities file emitting events to h.
// Both the returned diags slice and h.OnDiag carry parse diagnostics; they are identical.
func WalkEntities(src []byte, toks []scan.Token, h Handler, opts Opts) (diags []scan.Diagnostic) {
	c := &cursor{
		src:            src,
		toks:           toks,
		i:              -1,
		nToks:          len(toks),
		maxDiagnostics: opts.MaxDiagnostics,
	}
	c.walkEntities(h)
	return c.diags
}

// -------------------------
// 2) Cursor / token stream helper
// -------------------------

type cursor struct {
	src            []byte
	toks           []scan.Token
	i              int // -1 = before first token; i = index of last consumed token
	nToks          int
	diags          []scan.Diagnostic
	maxDiagnostics int  // 0 = unlimited
	truncated      bool // set once the cap is reached; suppresses further emissions
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

// diagSpan returns a zero-length span positioned just after the last consumed token.
func (c *cursor) diagSpan() scan.Span {
	if c.i >= 0 && c.i < c.nToks {
		end := c.toks[c.i].Span.End
		return scan.NewSpan(end, end)
	}
	return scan.NewSpan(0, 0)
}

// emitDiag records a diagnostic in c.diags and forwards it to h.
// Subject to the diagnostic-count cap configured on the cursor.
func (c *cursor) emitDiag(h Handler, d scan.Diagnostic) {
	c.recordDiag(h, d)
}

// appendDiag records a diagnostic in c.diags only — no handler forward.
// Subject to the diagnostic-count cap configured on the cursor.
func (c *cursor) appendDiag(d scan.Diagnostic) {
	c.recordDiag(nil, d)
}

// recordDiag is the cap-aware diagnostic appender. When the cap is reached it
// substitutes a single truncation sentinel and silences further calls; with
// no cap configured it appends unconditionally. Pass h=nil to skip forwarding.
func (c *cursor) recordDiag(h Handler, d scan.Diagnostic) {
	if c.truncated {
		return
	}
	if c.maxDiagnostics > 0 && len(c.diags) >= c.maxDiagnostics-1 {
		sentinel := scan.Diagnostic{
			Code:    Codes.DIAGNOSTICS_TRUNCATED,
			Span:    c.diagSpan(),
			Message: fmt.Sprintf("diagnostic limit (%d) reached; further diagnostics omitted", c.maxDiagnostics),
		}
		c.diags = append(c.diags, sentinel)
		if h != nil {
			h.OnDiag(sentinel)
		}
		c.truncated = true
		return
	}
	c.diags = append(c.diags, d)
	if h != nil {
		h.OnDiag(d)
	}
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
		c.appendDiag(scan.Diagnostic{Code: code, Span: c.diagSpan(), Message: msg})
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
		c.appendDiag(scan.Diagnostic{Code: code, Span: c.diagSpan(), Message: msg})
	}
	return tok, ok
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
)

type Value struct {
	Kind ValueKind
	Tok  scan.Token // for number/string/ident
	Obj  ObjectSpan // for object value
}

type ObjectSpan struct {
	LBraceTok scan.Token
	RBraceTok scan.Token
}

// -------------------------
// 4) Grammar / walk functions
// -------------------------

func (c *cursor) walkEntities(h Handler) {
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
			c.appendDiag(scan.Diagnostic{
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
		c.appendDiag(scan.Diagnostic{
			Code:    Codes.UNEXPECTED_TOKEN,
			Span:    c.diagSpan(),
			Message: "unexpected end of file in statement",
		})
		return
	}

	// Assignment: key = value [;]
	if next.Kind == scan.KindSymbol && c.src[next.Span.Start] == '=' {
		eqTok := c.next()
		val := c.parseValue(h)
		var semiTok scan.Token
		if val.Kind != ValObject {
			semi, ok := c.matchSym(';')
			if !ok {
				c.emitDiag(h, scan.Diagnostic{
					Code:    Codes.EXPECTED_SEMICOLON,
					Span:    c.diagSpan(),
					Message: "expected ';' after assignment",
				})
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
		c.appendDiag(scan.Diagnostic{
			Code:    Codes.UNEXPECTED_TOKEN,
			Span:    c.diagSpan(),
			Message: "expected value",
		})
		return Value{}
	}

	// Object value: { ... }
	if tok.Kind == scan.KindSymbol && c.src[tok.Span.Start] == '{' {
		lbrace := c.next()
		h.OnObjectBegin(*lbrace)
		c.walkObjectBody(h)
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
		h.OnObjectEnd(rbrace)
		return Value{Kind: ValObject, Obj: ObjectSpan{LBraceTok: *lbrace, RBraceTok: rbrace}}
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
