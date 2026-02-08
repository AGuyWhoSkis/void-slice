package scanner

import (
	"fmt"
	"void-slice/internal/scan"
)

// unimplemented
// right idea, wrong architecture
func Legacy_Lex(src []byte) (tokens []Token, diags []Diagnostic, err error) {
	n := len(src)
	peek := func(j int) (byte, bool) {
		if j >= n || j < 0 {
			return 0, false
		}
		return src[j], true
	}
	newToken := func(ntKind TokenKind, ntSpanStart int, ntSpanEnd int) Token {
		return Token{
			Kind: ntKind,
			Span: scan.Span{
				Start: ntSpanStart,
				End:   ntSpanEnd,
			},
		}
	}
	scanState := SCANNING_STANDARD
	nextToken := newToken(DECL_NIL, 0, 0)
	for b_offset := 0; b_offset < n; b_offset++ {
		b := src[b_offset]                // current byte
		bm1, ok_bm1 := peek(b_offset - 1) // prev byte

		switch scanState {
		case SCANNING_BLOCK_COMMENT:
			// look for */ that ends block comment, otherwise advance
			if b == '/' && ok_bm1 && bm1 == '*' {
				scanState = SCANNING_STANDARD
				nextToken.Span.End = b_offset
				tokens = append(tokens, nextToken)
			}
		case SCANNING_LINE_COMMENT:
			// look for \n that ends line comment, otherwise advance
			if b == '\n' {
				scanState = SCANNING_STANDARD
				nextToken.Span.End = b_offset
				tokens = append(tokens, nextToken)
			}
		case SCANNING_DOUBLE_QUOTE:
			// look for closing double quote
			if b == '"' && ok_bm1 && bm1 != '\\' { // unless escaped with \
				scanState = SCANNING_STANDARD
				nextToken.Span.End = b_offset
				tokens = append(tokens, nextToken)
			}
		case SCANNING_STANDARD:
			if IsWhitespace(b) {
				continue
			} else if b == '*' && ok_bm1 && bm1 == '/' { // look for  /* (block comment start)
				nextToken = newToken(DECL_BLOCK_COMMENT, b_offset-1, b_offset)
				scanState = SCANNING_BLOCK_COMMENT
			} else if b == '/' && ok_bm1 && bm1 == '/' { // look for line comment
				nextToken = newToken(DECL_LINE_COMMENT, b_offset-1, b_offset)
				scanState = SCANNING_LINE_COMMENT
			} else if b == '"' { // double quote
				nextToken = newToken(DECL_DOUBLE_QUOTE, b_offset, b_offset+1)
				scanState = SCANNING_DOUBLE_QUOTE
			} else if b == '{' {
				nextToken = newToken(DECL_BRACE_SQUIG_OPEN, b_offset, b_offset+1)
				tokens = append(tokens, nextToken)
			} else if b == '(' {
				nextToken = newToken(DECL_BRACE_ROUND_OPEN, b_offset, b_offset+1)
				tokens = append(tokens, nextToken)
			} else if b == '[' {
				nextToken = newToken(DECL_BRACE_SQUARE_OPEN, b_offset, b_offset+1)
				tokens = append(tokens, nextToken)
			} else if b == '}' {
				nextToken = newToken(DECL_BRACE_SQUIG_CLOSE, b_offset, b_offset+1)
				tokens = append(tokens, nextToken)
			} else if b == ')' {
				nextToken = newToken(DECL_BRACE_ROUND_CLOSE, b_offset, b_offset+1)
				tokens = append(tokens, nextToken)
			} else if b == ']' {
				nextToken = newToken(DECL_BRACE_SQUARE_CLOSE, b_offset, b_offset+1)
				tokens = append(tokens, nextToken)
			} else if b == '-' || b == '.' || b == '+' {

			} else if IsIdentStart(b) {
				nextToken.Span.Start += 1
				// if prev char is not ident, it must be time to add a new token
				// edit = {m_some2TestVarName="";}
				//                           ^
			} else if IsIdentCont(b) {

			} else if IsDigit(b) {

			} else {
				return tokens, diags, fmt.Errorf("fatal: unrecognized byte '%b'", b)
			}

			// drafted order:
			// whitespace skip
			// comment start detection (//, /)
			// punctuation (including bracket stack updates)
			// string start
			// identifier
			// number
			// fallback unknown
		default:
			panic(fmt.Errorf("invalid scan state %d", scanState))
		}
	}

	return tokens, diags, err
}

/*
func LexString(str string) (tokens []Token, diags []Diagnostic, err error)  // return Lex([]byte(str))
func LexReader(r io.Reader) (tokens []Token, diags []Diagnostic, err error) // ??
*/
