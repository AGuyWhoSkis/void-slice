package parse

import (
	"path/filepath"
	"strings"

	"void-slice/internal/scan"
)

// Shape identifies a top-level grammar variant for .decl / .entities files.
// See kanban/goals/M4-decl-taxonomy.md for the inventory and the dispatch rule.
type Shape int

const (
	ShapeEntities    Shape = iota // .entities → curly inherit/edit grammar (alias of Shape 1)
	Shape1Curly                   // .decl curly inherit/edit
	Shape2Animset                 // .decl previewmd6 / whitespace+quoted
	Shape3Material                // .decl m_PhysicsMaterial / tab-aligned
	Shape4Renderprog              // .decl newstyle / HLSL — permanent no-op (G-C carve-out)
	Shape5Md6def                  // .decl init { ... } (folds into Shape 2 grammar)
)

// Walk routes (path, src, toks) to the shape-specific walker and returns its
// diagnostics. Both the returned slice and h.OnDiag carry the diagnostics.
//
// Dispatch: extension picks .entities vs .decl; content-sniff picks among
// shapes 1–5. The lint layer owns the `.decl.xml` no-op classification, so
// XML files never reach this dispatcher. Shape 4 is a permanent no-op.
func Walk(path string, src []byte, toks []scan.Token, h Handler) []scan.Diagnostic {
	switch Classify(path, src, toks) {
	case Shape2Animset:
		return walkShape2(src, toks, h)
	case Shape3Material:
		return walkShape3(src, toks, h)
	case Shape4Renderprog:
		return walkShape4(src, toks, h)
	case Shape5Md6def:
		return walkShape5(src, toks, h)
	default:
		return WalkEntities(src, toks, h)
	}
}

// Classify applies the hybrid extension + content-sniff rule from
// kanban/goals/M4-decl-taxonomy.md § Dispatch decision.
func Classify(path string, src []byte, toks []scan.Token) Shape {
	if filepath.Ext(strings.ToLower(path)) != ".decl" {
		return ShapeEntities
	}
	return sniffDeclShape(src, toks)
}

func sniffDeclShape(src []byte, toks []scan.Token) Shape {
	i := skipComments(toks, 0)
	if i < len(toks) && toks[i].Kind == scan.KindSymbol && src[toks[i].Span.Start] == '{' {
		i = skipComments(toks, i+1)
	}
	if i >= len(toks) || toks[i].Kind != scan.KindIdentifier {
		return Shape1Curly
	}
	switch string(src[toks[i].Span.Start:toks[i].Span.End]) {
	case "previewmd6":
		return Shape2Animset
	case "m_PhysicsMaterial":
		return Shape3Material
	case "newstyle":
		return Shape4Renderprog
	case "init":
		return Shape5Md6def
	}
	return Shape1Curly
}

func skipComments(toks []scan.Token, i int) int {
	for i < len(toks) && (toks[i].Kind == scan.KindCommentLine || toks[i].Kind == scan.KindCommentBlock) {
		i++
	}
	return i
}

// walkShape5 is the md6def grammar; lexically near-identical to Shape 2 and
// folded into the same walker per the M4 taxonomy doc.
func walkShape5(src []byte, toks []scan.Token, h Handler) []scan.Diagnostic {
	return walkShape2(src, toks, h)
}

// walkShape4 is the permanent no-op walker for renderprog (G-C carve-out):
// shader source inside hlsl_prefix { ... } is not parsed.
func walkShape4(_ []byte, _ []scan.Token, _ Handler) []scan.Diagnostic { return nil }
