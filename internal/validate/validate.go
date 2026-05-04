package validate

import (
	"fmt"
	"strconv"

	"void-slice/internal/parse"
	"void-slice/internal/scan"
)

// ValidateEntities routes (path, src, toks) through parse.Walk and runs the
// semantic validator over the resulting events. Returned diagnostics include
// parse errors followed by validate warnings. Shape-stub walkers produce no
// events, so non-Shape-1 .decl files return only their (empty) parse diags.
func ValidateEntities(path string, src []byte, toks []scan.Token) []scan.Diagnostic {
	v := &validator{src: src}
	parseDiags := parse.Walk(path, src, toks, v)
	all := make([]scan.Diagnostic, 0, len(parseDiags)+len(v.diags))
	all = append(all, parseDiags...)
	all = append(all, v.diags...)
	return all
}

// objFrame tracks array-shape state for one nested object block.
type objFrame struct {
	numTok *scan.Token        // points at the num value token
	numVal *int64             // parsed value of num = N
	items  map[int64]scan.Token // index -> anchor token (indexer value tok)
}

type validator struct {
	src    []byte
	frames []objFrame
	diags  []scan.Diagnostic
}

func (v *validator) top() *objFrame {
	if len(v.frames) == 0 {
		return nil
	}
	return &v.frames[len(v.frames)-1]
}

func (v *validator) lexeme(tok scan.Token) []byte {
	return v.src[tok.Span.Start:tok.Span.End]
}

// Handler no-ops for events we don't need.
func (v *validator) OnVersion(versionTok scan.Token, versionValue int64)             {}
func (v *validator) OnComponentBegin(componentTok scan.Token, lbrace scan.Token)     {}
func (v *validator) OnComponentDecl(typeTok, nameTok scan.Token, lbrace scan.Token)  {}
func (v *validator) OnComponentEnd(rbrace scan.Token)                                 {}
func (v *validator) OnTypedBlock(typeTok, nameTok scan.Token, lbrace scan.Token)     {}
func (v *validator) OnDiag(diag scan.Diagnostic)                                      {}

func (v *validator) OnObjectBegin(lbrace scan.Token) {
	v.frames = append(v.frames, objFrame{items: make(map[int64]scan.Token)})
}

func (v *validator) OnAssignment(key parse.Key, eqTok scan.Token, value parse.Value, semiTok scan.Token) {
	f := v.top()
	if f == nil {
		return
	}
	base := string(v.lexeme(key.BaseTok))

	if base == "num" && value.Kind == parse.ValNumber {
		n, ok := parseIntLiteral(v.lexeme(value.Tok))
		if ok {
			tok := value.Tok
			f.numTok = &tok
			f.numVal = &n
		}
		return
	}

	if base == "item" && len(key.Indexers) == 1 && key.Indexers[0].Kind == parse.IndexInt {
		idx := key.Indexers[0].IntValue
		anchor := key.Indexers[0].ValueTok
		if _, dup := f.items[idx]; dup {
			v.diags = append(v.diags, scan.Diagnostic{
				Code:    Codes.ARRAY_DUP_INDEX,
				Span:    anchor.Span,
				Message: fmt.Sprintf("duplicate array index %d", idx),
			})
		} else {
			f.items[idx] = anchor
		}
	}
}

func (v *validator) OnObjectEnd(rbrace scan.Token) {
	f := v.top()
	if f == nil {
		return
	}
	defer func() { v.frames = v.frames[:len(v.frames)-1] }()

	if f.numVal != nil {
		expected := *f.numVal
		actual := int64(len(f.items))
		if actual != expected {
			v.diags = append(v.diags, scan.Diagnostic{
				Code:    Codes.ARRAY_COUNT_MISMATCH,
				Span:    f.numTok.Span,
				Message: fmt.Sprintf("array count mismatch: num=%d but %d item(s) defined", expected, actual),
			})
		}
		for idx, tok := range f.items {
			if idx < 0 || idx >= expected {
				v.diags = append(v.diags, scan.Diagnostic{
					Code:    Codes.ARRAY_INDEX_OOB,
					Span:    tok.Span,
					Message: fmt.Sprintf("array index %d out of bounds [0, %d)", idx, expected),
				})
			}
		}
	} else if len(f.items) > 0 {
		var firstTok scan.Token
		first := true
		for _, tok := range f.items {
			if first || tok.Span.Start < firstTok.Span.Start {
				firstTok = tok
				first = false
			}
		}
		v.diags = append(v.diags, scan.Diagnostic{
			Code:    Codes.ARRAY_MISSING_NUM,
			Span:    firstTok.Span,
			Message: "item[...] entries found but no num declaration",
		})
	}
}

func parseIntLiteral(b []byte) (int64, bool) {
	s := string(b)
	if len(s) > 0 && (s[len(s)-1] == 'f' || s[len(s)-1] == 'm') {
		s = s[:len(s)-1]
	}
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
