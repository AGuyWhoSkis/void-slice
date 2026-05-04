package parse

import "void-slice/internal/scan"

// walkShape3 handles the material (Shape 3) grammar: tab-aligned key-value
// pairs, no `=`, no `;`, with bare-path values (`models/.../arm.tga`),
// namespaced option keys (`wrinkles/enable`), function-call values
// (`ipr_constantcolor(0.6, 0.6, 0.6, 1)`), and brace-list tuple values
// (`wardRoughness { 0, 1, 0, 0 }`).
func walkShape3(src []byte, toks []scan.Token, h Handler) []scan.Diagnostic {
	c := &cursor{src: src, toks: toks, i: -1, nToks: len(toks)}
	c.walkShape3Top(h)
	return c.diags
}

func (c *cursor) walkShape3Top(h Handler) {
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
		c.walkShape3Body(h)
		rbrace, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close top-level object")
		if !ok {
			c.syncTo('}', 0)
			rbrace, _ = c.matchSym('}')
		}
		h.OnObjectEnd(rbrace)
		return
	}
	c.walkShape3Body(h)
}

func (c *cursor) walkShape3Body(h Handler) {
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
			c.walkShape3Statement(h)
		default:
			c.emitDiag(h, scan.Diagnostic{
				Code:    Codes.UNEXPECTED_TOKEN,
				Span:    t.Span,
				Message: "unexpected token in shape-3 body",
			})
			c.next()
		}
	}
}

func (c *cursor) walkShape3Statement(h Handler) {
	keyTok := c.next() // IDENT

	// Namespaced key: IDENT (/ IDENT)+ (e.g. `materialEffects/enable`).
	for c.matchShape3SlashIdent() {
	}

	t := c.peek()
	if t == nil {
		return
	}

	// Brace value: either a bare typed block or a brace-list of numbers.
	if t.Kind == scan.KindSymbol && c.src[t.Span.Start] == '{' {
		lbrace := c.next()
		next := c.peek()
		if next != nil && next.Kind == scan.KindNumberLiteral {
			// Brace-list tuple value (`{ 0, 1, 0, 0 }`). Consume opaquely.
			c.consumeUntilRBrace()
			c.matchSym('}')
			return
		}
		h.OnTypedBlock(*keyTok, scan.Token{}, *lbrace)
		c.walkShape3Body(h)
		if _, ok := c.expectSym('}', Codes.EXPECTED_SYMBOL, "expected '}' to close shape-3 typed block"); !ok {
			c.syncTo('}', 0)
			c.matchSym('}')
		}
		return
	}

	switch t.Kind {
	case scan.KindQuoteLiteral:
		tok := c.next()
		h.OnAssignment(Key{BaseTok: *keyTok}, scan.Token{}, Value{Kind: ValString, Tok: *tok}, scan.Token{})
	case scan.KindNumberLiteral:
		tok := c.next()
		h.OnAssignment(Key{BaseTok: *keyTok}, scan.Token{}, Value{Kind: ValNumber, Tok: *tok}, scan.Token{})
	case scan.KindIdentifier:
		// Bare-path or function-call value: IDENT (/ IDENT)* (. IDENT)* and
		// optionally a parenthesized arg list. Consume the full run; anchor
		// the emitted value at the first IDENT.
		first := c.next()
		c.consumeShape3PathTail()
		if p := c.peek(); p != nil && p.Kind == scan.KindSymbol && c.src[p.Span.Start] == '(' {
			c.next() // (
			c.consumeUntilRParen()
			c.matchSym(')')
		}
		h.OnAssignment(Key{BaseTok: *keyTok}, scan.Token{}, Value{Kind: ValIdent, Tok: *first}, scan.Token{})
	}
}

// matchShape3SlashIdent consumes `/ IDENT` if the lookahead matches.
func (c *cursor) matchShape3SlashIdent() bool {
	t1 := c.peek()
	if t1 == nil || t1.Kind != scan.KindSymbol || c.src[t1.Span.Start] != '/' {
		return false
	}
	if c.i+2 >= c.nToks {
		return false
	}
	t2 := &c.toks[c.i+2]
	if t2.Kind != scan.KindIdentifier {
		return false
	}
	c.next() // /
	c.next() // IDENT
	return true
}

// consumeShape3PathTail consumes zero or more `(/ IDENT | . IDENT)` segments
// after a leading IDENT in a value position (bare-path values like
// `models/foo/bar.tga`).
func (c *cursor) consumeShape3PathTail() {
	for {
		t1 := c.peek()
		if t1 == nil || t1.Kind != scan.KindSymbol {
			return
		}
		b := c.src[t1.Span.Start]
		if b != '/' && b != '.' {
			return
		}
		if c.i+2 >= c.nToks {
			return
		}
		t2 := &c.toks[c.i+2]
		if t2.Kind != scan.KindIdentifier && t2.Kind != scan.KindNumberLiteral {
			return
		}
		c.next() // / or .
		c.next() // IDENT or NUMBER
	}
}

// consumeUntilRBrace consumes tokens until (but not including) a matching
// `}`. Nested `{` is honored. Used for opaque brace-list tuple values.
func (c *cursor) consumeUntilRBrace() {
	depth := 1
	for {
		t := c.peek()
		if t == nil {
			return
		}
		if t.Kind == scan.KindSymbol {
			b := c.src[t.Span.Start]
			if b == '{' {
				depth++
			} else if b == '}' {
				depth--
				if depth == 0 {
					return
				}
			}
		}
		c.next()
	}
}

// consumeUntilRParen consumes tokens until (but not including) a matching
// `)`. Nested `(` is honored. Used for opaque function-call argument lists.
func (c *cursor) consumeUntilRParen() {
	depth := 1
	for {
		t := c.peek()
		if t == nil {
			return
		}
		if t.Kind == scan.KindSymbol {
			b := c.src[t.Span.Start]
			if b == '(' {
				depth++
			} else if b == ')' {
				depth--
				if depth == 0 {
					return
				}
			}
		}
		c.next()
	}
}
