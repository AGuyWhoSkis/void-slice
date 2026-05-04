package parse

import "void-slice/internal/scan"

// walkShape2 handles the animset (Shape 2) and md6def (Shape 5) grammars,
// which are lexically near-identical: curly-braced root, no `=`, no `;`,
// whitespace-separated key-value pairs with quoted-string values, integer
// literals, and nested object blocks. Per the M4 taxonomy doc the two shapes
// share one walker.
//
// Statement forms inside a body:
//   - key { body }           → OnTypedBlock(key, zero, lbrace)
//   - key NUMBER { body }    → OnTypedBlock(key, numTok, lbrace)
//   - key "STRING" { body }  → OnTypedBlock(key, strTok, lbrace)
//   - key value              → OnAssignment(key, zero, value, zero)
//   - key v1 v2 v3 ...       → OnAssignment(key, zero, v1, zero); v2+ silent
//   - "string" or NUMBER     → consumed silently (group-list payload)
func walkShape2(src []byte, toks []scan.Token, h Handler) []scan.Diagnostic {
	c := &cursor{src: src, toks: toks, i: -1, nToks: len(toks)}
	c.walkShape2Top(h)
	return c.diags
}

func (c *cursor) walkShape2Top(h Handler) {
	// Outer `{ ... }` wrapper — same shape as Shape-1 bare-curly.
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
		c.walkShape2Body(h)
		rbrace, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close top-level object")
		if !ok {
			c.syncTo('}', 0)
			rbrace, _ = c.matchSym('}')
		}
		h.OnObjectEnd(rbrace)
		return
	}
	// No outer brace — walk loose statements.
	c.walkShape2Body(h)
}

func (c *cursor) walkShape2Body(h Handler) {
	for {
		t := c.peek()
		if t == nil {
			return
		}
		if t.Kind == scan.KindSymbol && c.src[t.Span.Start] == '}' {
			return
		}
		switch t.Kind {
		case scan.KindCommentLine, scan.KindCommentBlock:
			c.next()
		case scan.KindIdentifier:
			c.walkShape2Statement(h)
		case scan.KindQuoteLiteral, scan.KindNumberLiteral:
			// Standalone scalar inside a body (e.g. `groups { "animations" }`,
			// list payloads). No key, no event — just consume.
			c.next()
		default:
			c.emitDiag(h, scan.Diagnostic{
				Code:    Codes.UNEXPECTED_TOKEN,
				Span:    t.Span,
				Message: "unexpected token in shape-2 body",
			})
			c.next()
		}
	}
}

func (c *cursor) walkShape2Statement(h Handler) {
	keyTok := c.next() // IDENT

	// Scoped name continuation: `arkMidnightAnimMarker::Name_t` reads as
	// IDENT (:: IDENT)+. Consume the trailing segments silently — the key
	// span stays anchored on the first IDENT.
	for {
		t1 := c.peek()
		if t1 == nil || t1.Kind != scan.KindSymbol || c.src[t1.Span.Start] != ':' {
			break
		}
		if c.i+2 >= c.nToks {
			break
		}
		t2 := &c.toks[c.i+2]
		t3 := &c.toks[c.i+3]
		if t2.Kind != scan.KindSymbol || c.src[t2.Span.Start] != ':' || t3.Kind != scan.KindIdentifier {
			break
		}
		c.next() // :
		c.next() // :
		c.next() // IDENT
	}

	var primary Value
	have := false

	for {
		t := c.peek()
		if t == nil {
			break
		}
		if t.Kind == scan.KindCommentLine || t.Kind == scan.KindCommentBlock {
			c.next()
			continue
		}
		if t.Kind == scan.KindSymbol && c.src[t.Span.Start] == '{' {
			lbrace := c.next()
			var nameTok scan.Token
			if have && (primary.Kind == ValNumber || primary.Kind == ValString || primary.Kind == ValIdent) {
				nameTok = primary.Tok
			}
			h.OnTypedBlock(*keyTok, nameTok, *lbrace)
			c.walkShape2Body(h)
			if _, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close shape-2 typed block"); !ok {
				c.syncTo('}', 0)
				c.matchSym('}')
			}
			return
		}
		if t.Kind == scan.KindQuoteLiteral {
			tok := c.next()
			if !have {
				primary = Value{Kind: ValString, Tok: *tok}
				have = true
			}
			continue
		}
		if t.Kind == scan.KindNumberLiteral {
			tok := c.next()
			if !have {
				primary = Value{Kind: ValNumber, Tok: *tok}
				have = true
			}
			continue
		}
		if t.Kind == scan.KindIdentifier {
			if !have {
				tok := c.next()
				primary = Value{Kind: ValIdent, Tok: *tok}
				have = true
				continue
			}
			// Next statement starts here — leave it for the body loop.
			break
		}
		// Any other symbol (including `}`) ends the statement.
		break
	}

	if have {
		h.OnAssignment(Key{BaseTok: *keyTok}, scan.Token{}, primary, scan.Token{})
	}
}
