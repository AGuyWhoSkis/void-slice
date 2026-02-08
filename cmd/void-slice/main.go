package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const PATH_VF_D2 = "./void-files/VoidComponents-D2-all-text/Dishonored2/Export/game0"
const PATH_VF_DOTO = "./void-files/VoidComponents-DOTO-all-text/Dishonored_DeathOfTheOutsider/Export/game0"

func main() {
	selector := "generated.decls.localized.english.speechbarks.speech.speech_barks.special.duke.bk_duke_search.decl"

	// for now, tests are enough to develop against
	// TODO: eventually make main() do something useful
	path := filepath.Join(PATH_VF_D2, selector)
	read, errRead := os.ReadFile(path)
	if errRead != nil {
		fmt.Println(fmt.Errorf("io error: %v", errRead))
	}

	fmt.Println(read)
}

/*

one canonical parse entry point:


and then layer wrappers:

ParseBytes(b []byte) → calls Parse(bytes.NewReader(b))

ParseString(s string) → calls Parse(strings.NewReader(s))

*/

type ErrorType string

const (
	MISMATCH_QUOTE = "MISMATCH QUOTE"
)

// func Parse(r io.Reader) ([]lex.Token, error) {
// 	br := bufio.NewReader(r)

// 	var stackPairedChars []lex.Token

// 	line, col := 1, 0
// 	lineSoFar := ""
// 	debugMarks := []mark{}
// 	bIsWithinDblQuote := false
// 	bIsWithinSngQuote := false
// 	parsingErrors := []ParseError{}

// 	for {
// 		b, err := br.ReadByte()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			return nil, err
// 		}
// 		col++

// 		switch b {
// 		case '(', '[', '{':
// 			if bIsWithinDblQuote || bIsWithinSngQuote {
// 				parsingErrors = append(parsingErrors, ParseError{
// 					// TODO: test against game files to answer 'are there any valid quotes containing brackets?'
// 					Line:  line,
// 					Col:   col,
// 					Error: fmt.Errorf("illegal? %q within quote at %d:%d\n\tLSF: %s", b, line, col, lineSoFar),
// 				})
// 				// pretend the quote is closed and move on
// 				bIsWithinDblQuote = false
// 				bIsWithinSngQuote = false
// 			}
// 			stackPairedChars = append(stackPairedChars, newToken(b, line, col))
// 		case ')', ']', '}':
// 			if bIsWithinDblQuote || bIsWithinSngQuote {
// 				// syntax err: i have yet to see a quote containing a closing bracket
// 				parsingErrors = append(parsingErrors, ParseError{
// 					Line:  line,
// 					Col:   col,
// 					Error: fmt.Errorf("illegal? %q within quote at %d:%d\n\tLSF: %s", b, line, col, lineSoFar),
// 				})
// 				// pretend the quote is closed and move on
// 				bIsWithinDblQuote = false
// 				bIsWithinSngQuote = false
// 			}
// 			if len(stackPairedChars) == 0 {
// 				return nil, fmt.Errorf("unexpected %q at %d:%d\n\tLSF: %s", b, line, col, lineSoFar)
// 			}
// 			top := stackPairedChars[len(stackPairedChars)-1]
// 			if !matches(top.Ch, b) {
// 				parsingErrors = append(parsingErrors, ParseError{
// 					Line: line,
// 					Col:  col,
// 					Error: fmt.Errorf("mismatched: opened %q at %d:%d, got %q at %d:%d\n\tLSF: %s",
// 						top.Ch, top.Pos.Line, top.Pos.Col, b, line, col, lineSoFar),
// 				})
// 				// assume it's recoverable
// 				// TODO reset the state by inserting matching bracket onto stack

// 				// requiredPair := pairOf(top.ch)

// 				stackPairedChars = stackPairedChars[:len(stackPairedChars)-1]
// 				topPairCh := pairOf(top.Ch)
// 				topPair := lex.Token{
// 					Ch: topPairCh,
// 					Pos: lex.Pos{
// 						Line: top.Pos.Line,
// 						Col:  top.Pos.Col - 1,
// 					},
// 				}
// 				stackPairedChars = append(stackPairedChars, topPair, top)
// 			}
// 			stackPairedChars = stackPairedChars[:len(stackPairedChars)-1]
// 		case '"': // "Double Quotes"
// 			if bIsWithinSngQuote {
// 				continue
// 			} else if len(stackPairedChars) > 0 {
// 				top := stackPairedChars[len(stackPairedChars)-1]
// 				if matches(top.Ch, b) {
// 					// remove from stack
// 					stackPairedChars = stackPairedChars[:len(stackPairedChars)-1]
// 					bIsWithinDblQuote = false
// 					// debugMarks = append(debugMarks, mark{line: line, col: col, displayAs: "|" + string(top.ch) + "|" + string(b)})
// 					continue
// 				}
// 			}
// 			// otherwise add to stack
// 			stackPairedChars = append(stackPairedChars, newToken(b, line, col))
// 			bIsWithinDblQuote = true
// 			// debugMarks = append(debugMarks, mark{line: line, col: col, displayAs: "0" + string(top.ch) + "0" + string(b)})
// 		case '\'': // 'Single Quotes'
// 			if bIsWithinDblQuote {
// 				continue
// 			} else if len(stackPairedChars) > 0 {
// 				top := stackPairedChars[len(stackPairedChars)-1]
// 				if matches(top.Ch, b) {
// 					// remove from stack
// 					stackPairedChars = stackPairedChars[:len(stackPairedChars)-1]
// 					bIsWithinSngQuote = false
// 					// debugMarks = append(debugMarks, mark{line: line, col: col, displayAs: "|" + string(top.ch) + "|" + string(b)})
// 					continue
// 				}
// 			}
// 			// otherwise add to stack
// 			stackPairedChars = append(stackPairedChars, newToken(b, line, col))
// 			bIsWithinSngQuote = true
// 			// debugMarks = append(debugMarks, mark{line: line, col: col, displayAs: "0" + string(top.ch) + "0" + string(b)})
// 		}

// 		if b == '\n' {
// 			if bIsWithinDblQuote {
// 				return nil, fmt.Errorf("no matching pair for double-quote (\") on line %d\n\tLSF: %s", line, lineSoFar)
// 			} else if bIsWithinSngQuote {
// 				return nil, fmt.Errorf("no matching pair for single-quote (') on line %d\n\tLSF: %s", line, lineSoFar)
// 			}

// 			fmt.Println(lineSoFar)
// 			if len(debugMarks) > 0 {
// 				markings := ""

// 				for i := 0; i < col; i++ {
// 					isMarkByCol := func(m mark) bool {
// 						return i == m.col
// 					}
// 					index := slices.IndexFunc(debugMarks, isMarkByCol)

// 					colIsMarked := index != -1

// 					if colIsMarked {
// 						m := debugMarks[index]
// 						markings = markings + string(m.displayAs)
// 					} else {
// 						markings = markings + string(lineSoFar[i])
// 					}
// 				}
// 				fmt.Println(markings)
// 				debugMarks = []mark{}
// 			}
// 			lineSoFar = ""
// 			col = 0
// 			line++
// 		} else {
// 			lineSoFar = lineSoFar + string(b)
// 		}

// 	}

// 	if len(stackPairedChars) != 0 {
// 		top := stackPairedChars[len(stackPairedChars)-1]
// 		return nil, fmt.Errorf("unclosed %q opened at %d:%d", top.Ch, top.Pos.Line, top.Pos.Col)
// 	}

// 	return stackPairedChars, nil
// }

// func newToken(ch byte, line, col int) lex.Token {
// 	return lex.Token{
// 		Ch:  ch,
// 		Pos: lex.Pos{Line: line, Col: col},
// 	}
// }

func matches(open, close byte) bool {
	switch open {
	case '(':
		return close == ')'
	case '[':
		return close == ']'
	case '{':
		return close == '}'
	case '<':
		return close == '>'
	case '"', '\'':
		return open == close
	}

	panic(fmt.Errorf("oh FUCK %b:%b", open, close))
}

func pairOf(openOrClose byte) byte {
	switch openOrClose {
	case '\'':
		return '\''
	case '"':
		return '"'
	case '(':
		return ')'
	case ')':
		return '('
	case '{':
		return '}'
	case '}':
		return '{'
	case '[':
		return ']'
	case '<':
		return '>'
	case '>':
		return '<'
	}
	panic(fmt.Errorf("no pair for char %b", openOrClose))
}
