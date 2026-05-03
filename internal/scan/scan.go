package scan

// Planned: keep the scanner stable; only add tiny helpers if they reduce
// friction in parse/validate without changing the token model
//
// Goals:
//   - Scanner stays: []byte -> []Token{Kind+Span} + []Diagnostic
//   - Parser/Validator distinguishes punctuation via src[t.Span.Start] (Option A)
//
// Optional tiny additions (only if friction appears):
//   [ ] Add helper to avoid repeating span slicing everywhere:
//         func Lexeme(src []byte, t Token) []byte { return src[t.Span.Start:t.Span.End] }
//       (Place in scan_util.go; safe, no allocations beyond slice header)
//
//   [ ] Add helper for single-byte symbol extraction (assert length==1 if desired):
//         func Sym(src []byte, t Token) byte { return src[t.Span.Start] }
//
// Non-goals:
//   - Do NOT add keyword kinds (Version/component/edit/num/etc)
//   - Do NOT split SYMBOL kinds yet (back-pocket option to simplify bracket matching)
//   - Do NOT introduce parse coupling here (keep scan reusable)

import "fmt"

// Input:
//   - []byte from any source
//
// Output:
//
//   - []Token with Span{Start, End} and Kind
//
//   - diagnostics for lexical problems only
//
//   - possibly “fatal” errors only for IO, out of memory, etc.
//
//   - Output contract:
//
//   - Tokens are in order.
//
//   - Spans are half-open: [Start, End).
//
//   - The scanner never panics on malformed input; it emits diags and tries to recover.
//
// see 'TokenKind' (scan_constants.go) for a list of all Token.Kind values
func Scan(src []byte) (tokens []Token, diags []Diagnostic, newlineIndexes []int) {
	n := len(src)

	// helper for appending to 'diags'
	emitDiag := func(c DiagnosticCode, start, end int, msg string, args ...interface{}) {
		if len(args) > 0 {
			msg = fmt.Sprint(msg, args)
		}
		diags = append(diags, Diagnostic{Code: c, Span: NewSpan(start, end), Message: msg})
	}

	// helper for appending to 'tokens'
	emitToken := func(kind Kind, startOffset, endOffset int) {
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

mainLoop:
	for i := 0; i < n; i++ {
		b := src[i]
		switch b {

		case ' ', '\t', '\r':
			continue

		case '\n':
			newlineIndexes = append(newlineIndexes, i)
		case '"':
			if i+1 >= n {
				emitToken(TokenKind.QUOTE_LITERAL, i, i+1)
				break
			}
			for j := i + 1; j < n; j++ {
				j_b := src[j]
				if j_b == '\\' { // handle backslash escape '\' inside dbl. quote: ignore whatever the next char is
					j += 1
					continue
				} else if j_b == '"' {
					// end quote
					emitToken(TokenKind.QUOTE_LITERAL, i, j+1)
					i = j
					continue mainLoop
				}
			}
			// fall-through case: at this point, quote literal has reached EOF
			emitToken(TokenKind.QUOTE_LITERAL, i, n)
			emitDiag(Codes.SCAN, i, n, "unterminated quote")
			i = n
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if i+1 >= n {
				emitToken(TokenKind.NUMBER_LITERAL, i, i+1)
				break
			}
			for i_numLitr := i + 1; i_numLitr < n; i_numLitr++ {
				b_numLitr := src[i_numLitr]
				if IsDigit(b_numLitr) || IsNumberTypeSuffix(b_numLitr) || b_numLitr == '.' {
					continue
				} else {
					emitToken(TokenKind.NUMBER_LITERAL, i, i_numLitr)
					i = i_numLitr - 1
					continue mainLoop
				}
			}
			// fall-through case: at this point, number literal has reached EOF
			emitToken(TokenKind.NUMBER_LITERAL, i, n)
			emitDiag(Codes.SCAN, i, n, "unterminated number literal")
			i = n

		case '{', '}', '[', ']', '=', ';', ',', '(', ')', ':':
			emitToken(TokenKind.SYMBOL, i, i+1)

		default:
			b_next, ok_bNext := peek(i + 1)
			if b == '/' && ok_bNext && b_next == '*' {
				// parse block quote
				for i_blkQuot := i + 1; i_blkQuot < n; i_blkQuot++ {
					b_blkQuot := src[i_blkQuot]
					b_hasNext := i_blkQuot+1 < n
					if b_blkQuot == '*' && b_hasNext && src[i_blkQuot+1] == '/' {
						emitToken(TokenKind.COMMENT_BLOCK, i, i_blkQuot+1)
						i = i_blkQuot + 1 // peek-ahead byte '*' adds extra +1 here
						continue mainLoop
					}
				}
			} else if b == '/' && ok_bNext && b_next == '/' {
				// parse line comment, emit token
				for i_lineCmt := i + 2; i_lineCmt < n; i_lineCmt++ {
					b_lineCmt := src[i_lineCmt]
					if b_lineCmt == '\n' {
						// CRLF: exclude trailing '\r' from the comment span
						end := i_lineCmt
						if end > i+2 && src[end-1] == '\r' {
							end--
						}
						emitToken(TokenKind.COMMENT_LINE, i, end)
						i = i_lineCmt
						newlineIndexes = append(newlineIndexes, i_lineCmt)
						continue mainLoop
					}
				}
			} else if IsIdentStart(b) {
				// scan until IsIdentCont() is false
				for i_ident := i + 1; i_ident < n; i_ident++ {
					b_identCont := src[i_ident]
					if !IsIdentCont(b_identCont) {
						emitToken(TokenKind.IDENTIFIER, i, i_ident)
						i = i_ident - 1 // same as NUMBER_LITERAL: let outer i++ land on the delimiter
						continue mainLoop
					}
				}
				// identifier reached EOF
				emitToken(TokenKind.IDENTIFIER, i, n)
				i = n
			} else {
				// produce a Diagnostic that this char could not be parsed
				emitDiag(Codes.SCAN, i, i+1, fmt.Sprintf("unknown byte %b", b))
			}
		}
	}

	return tokens, diags, newlineIndexes
}
