package lsp

import (
	"void-slice/internal/lint"
	"void-slice/internal/scan"
)

func spanToRange(li scan.LineIndex, sp scan.Span) Range {
	start := li.PosAt(sp.Start)
	end := li.PosAt(sp.End)
	return Range{
		Start: Position{Line: start.Line - 1, Character: start.Col - 1},
		End:   Position{Line: end.Line - 1, Character: end.Col - 1},
	}
}

func convertDiagnostics(src []byte, ds []lint.Diagnostic) []Diagnostic {
	out := make([]Diagnostic, len(ds))
	li := scan.BuildLineIndex(src)
	for i, d := range ds {
		sev := SeverityError
		if d.Severity == lint.Warning {
			sev = SeverityWarning
		}
		out[i] = Diagnostic{
			Range:    spanToRange(li, d.Span),
			Severity: sev,
			Code:     d.Code,
			Source:   "voidslice",
			Message:  d.Message,
		}
	}
	return out
}
