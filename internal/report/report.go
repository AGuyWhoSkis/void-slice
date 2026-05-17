package report

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"void-slice/internal/scan"
)

type RenderOptions struct {
	ContextLines int
	UseColor     bool // TODO: apply ANSI escapes when opt.UseColor (for T5)
}

// MaxRenderedDiagnostics caps both pretty and JSON output. Defense-in-depth
// against any future cascade (engine fixes close known ones at the parser
// layer; this constant guarantees the user-visible surface stays bounded
// regardless of emitter behavior). 500 was chosen as "enough to surface the
// earliest problems on a screenful per render iteration, small enough that
// a corrupted file's JSON stays kilobyte-scale crossing the WASM boundary."
const MaxRenderedDiagnostics = 500

func Render(filename string, src []byte, diags []scan.Diagnostic, opt RenderOptions) string {
	if len(diags) == 0 {
		return ""
	}
	sorted := sortDiags(diags)
	truncated := 0
	if len(sorted) > MaxRenderedDiagnostics {
		truncated = len(sorted) - MaxRenderedDiagnostics
		sorted = sorted[:MaxRenderedDiagnostics]
	}
	li := scan.BuildLineIndex(src)

	var b strings.Builder
	for i, d := range sorted {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(renderOne(filename, src, li, d))
	}
	if truncated > 0 {
		b.WriteByte('\n')
		b.WriteString("... and ")
		b.WriteString(strconv.Itoa(truncated))
		b.WriteString(" more diagnostics suppressed (render cap: ")
		b.WriteString(strconv.Itoa(MaxRenderedDiagnostics))
		b.WriteString(")")
	}
	return b.String()
}

func RenderJSON(filename string, src []byte, diags []scan.Diagnostic) string {
	li := scan.BuildLineIndex(src)
	sorted := sortDiags(diags)
	truncated := 0
	if len(sorted) > MaxRenderedDiagnostics {
		truncated = len(sorted) - MaxRenderedDiagnostics
		sorted = sorted[:MaxRenderedDiagnostics]
	}

	jd := make([]jsonDiag, len(sorted))
	for i, d := range sorted {
		sp := li.SpanPos(d.Span)
		jd[i] = jsonDiag{
			Line:     sp.Start.Line,
			Col:      sp.Start.Col,
			Severity: severityLabel(d.Code),
			Code:     string(d.Code),
			Message:  d.Message,
		}
	}

	out, _ := json.MarshalIndent(jsonReport{File: filename, Diagnostics: jd, Truncated: truncated}, "", "  ")
	return string(out)
}

// warningCodes lists non-`VALIDATE_*` codes that render as warnings. The
// `lint` layer owns the severity classification; this set keeps the report
// layer in sync. Add to it when a new warning-class code lands outside the
// VALIDATE_ family.
var warningCodes = map[scan.DiagnosticCode]bool{
	"LINT_VE_INCONSISTENCY": true,
}

func severityLabel(code scan.DiagnosticCode) string {
	if strings.HasPrefix(string(code), "VALIDATE_") || warningCodes[code] {
		return "warning"
	}
	return "error"
}

func sortDiags(diags []scan.Diagnostic) []scan.Diagnostic {
	out := make([]scan.Diagnostic, len(diags))
	copy(out, diags)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Span.Start != out[j].Span.Start {
			return out[i].Span.Start < out[j].Span.Start
		}
		return string(out[i].Code) < string(out[j].Code)
	})
	return out
}

// extractLine returns the bytes of line lineNum (1-based) and its [lineStart, lineEnd)
// byte range within src. lineEnd is exclusive and stops before the '\n'.
func extractLine(src []byte, li scan.LineIndex, lineNum int) (text []byte, lineStart, lineEnd int) {
	if lineNum == 1 {
		lineStart = 0
	} else {
		lineStart = li.Newlines[lineNum-2] + 1
	}
	if lineNum-1 < len(li.Newlines) {
		lineEnd = li.Newlines[lineNum-1]
	} else {
		lineEnd = len(src)
	}
	return src[lineStart:lineEnd], lineStart, lineEnd
}

func renderOne(filename string, src []byte, li scan.LineIndex, d scan.Diagnostic) string {
	sp := li.SpanPos(d.Span)
	line := sp.Start.Line
	col := sp.Start.Col

	lineText, lineStart, lineEnd := extractLine(src, li, line)

	lineNumStr := strconv.Itoa(line)
	prefix := "  " + lineNumStr + " | "

	caretStart := d.Span.Start - lineStart
	caretLen := d.Span.End - d.Span.Start
	if maxLen := lineEnd - d.Span.Start; caretLen > maxLen {
		caretLen = maxLen
	}
	if caretLen <= 0 {
		caretLen = 1
	}

	header := filename + ":" + lineNumStr + ":" + strconv.Itoa(col) +
		" " + severityLabel(d.Code) + " [" + string(d.Code) + "] " + d.Message

	var b strings.Builder
	b.WriteString(header)
	b.WriteByte('\n')
	b.WriteString(prefix)
	b.Write(lineText)
	b.WriteByte('\n')
	b.WriteString(strings.Repeat(" ", len(prefix)+caretStart))
	b.WriteString(strings.Repeat("^", caretLen))
	return b.String()
}

type jsonDiag struct {
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type jsonReport struct {
	File        string     `json:"file"`
	Diagnostics []jsonDiag `json:"diagnostics"`
	Truncated   int        `json:"truncated,omitempty"`
}
