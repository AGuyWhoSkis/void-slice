package scan

// Input:
// 	- Source.Bytes ([]byte)
// Output:
// 	- []Token with Span{Start, End} and Kind
// 	- diagnostics for lexical problems only
// 	- possibly “fatal” errors only for IO, out of memory, etc.

// 	- Output contract:
// 	- Tokens are in order.
// 	- Spans are half-open: [Start, End).
// 	- The scanner never panics on malformed input; it emits diags and tries to recover.

func Scan(src []byte) (tokens []Token, diags []Diagnostic) {
	n := len(src)

	emit := func(kind Kind, startOffset, endOffset int) {
		tokens = append(tokens, Token{
			Kind: kind,
			Span: NewSpan(startOffset, endOffset),
		})
	}
	peek := func(j int) (byte, bool) {
		if j >= n || j < 0 {
			return 0, false
		}
		return src[j], true
	}

	// var kindOfToken *TokenKind = nil

	// Token kinds needed for Stage 2:
	//	 	- punctuation: { } [ ] ( ) = ; , (maybe . depending)
	// 		- identifiers: edit, m_rules, enumItem, item, num, m_name, etc
	//	 	- number literals: 34, -1, 23.89 (float maybe)
	//	 	- string literals: "..." (with escapes)
	// 		- comment tokens: line and block (or simply skip them)
	// 		- newlines/whitespace usually skipped
	for i := 0; i < n; i++ {
		b := src[i]
		switch b {
		case ' ', '\t':
			continue
		case '"':
			// quote literal
			for j := i + 1; j < n; j++ {
				j_b := src[j]
				if j_b == '\\' { // escape \ byte
					j += 1
					continue
				} else if j_b == '"' {
					// end of quote
					emit(TokenKind.QUOTE_LITERAL, i, j)
					break
				}
			}
		case '{', '}', '[', ']', '=', ';':
			// symbol
			emit(TokenKind.SYMBOL, i, i+1)
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			// number literal
			// known 'continue' bytes: 0123456789.mf
			if i+1 >= n {
				emit(TokenKind.NUMBER_LITERAL, i, i+1)
			}
			for i_numLitr := i + 1; i_numLitr < n; i_numLitr++ {
				b_numLitr := src[i_numLitr]
				if b_numLitr == '.' || b_numLitr == 'm' || b_numLitr == 'f' || IsDigit(b_numLitr) {
					continue
				}
				emit(TokenKind.NUMBER_LITERAL, i, i_numLitr)
			}
		default:
			b_next, ok_bNext := peek(i + 1)
			if !ok_bNext {
				// simplifying edge cases
				continue
			}

			if b == '/' && b_next == '*' {
				// parse block quote
				for i_blkQuot := i + 1; i_blkQuot < n; i_blkQuot++ {
					b_blkQuot := src[i_blkQuot]
					b_hasNext := i_blkQuot+1 < n
					if b_blkQuot == '*' && b_hasNext && src[i_blkQuot+1] == '/' {
						emit(TokenKind.COMMENT_BLOCK, i, i_blkQuot+2)
						i += i_blkQuot + 2 // no clue if this is right but I think so
						// TODO: push token
					}
				}

			} else if b == '/' && b_next == '/' {
				// parse line quote, emit token

			}

		}
	}

	return nil, nil
}

// 	Lexical diagnostics:
// 		- unterminated string
// 		- unterminated block comment
// 		- invalid escape sequence (if you enforce)
//	 	- invalid byte / unexpected control chars (optional)

// Recovery policy at this stage:
// 		- Unterminated string: emit diag at start, then treat rest-of-file as string token or bail scanning (I’d prefer “emit diag + end token at EOF and continue no further”).
// 		- Unterminated block comment similarly.
// 		- Unknown byte: emit diag and skip 1 byte.

// This stage should NOT attempt:
//  	 - bracket parity as a “correctness” decision (it can emit brace tokens; later stage checks balancing)
//  	 - num vs item[] counts (needs structure)
//  	 - “field mismatch causes crash” (semantic / schema)
