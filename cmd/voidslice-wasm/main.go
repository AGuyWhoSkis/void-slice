//go:build js && wasm

package main

import (
	"syscall/js"

	"void-slice/internal/lint"
	"void-slice/internal/report"
	"void-slice/internal/scan"
)

func main() {
	js.Global().Set("voidsliceLint", js.FuncOf(lintFn))
	select {}
}

func lintFn(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return errorJSON("voidsliceLint requires (filename, src)")
	}
	filename := args[0].String()
	src := []byte(args[1].String())

	diags, err := lint.New().Lint(filename, src)
	if err != nil {
		return errorJSON(err.Error())
	}

	scanDiags := make([]scan.Diagnostic, len(diags))
	for i, d := range diags {
		scanDiags[i] = scan.Diagnostic{
			Code:    scan.DiagnosticCode(d.Code),
			Span:    d.Span,
			Message: d.Message,
		}
	}
	return report.RenderJSON(filename, src, scanDiags)
}

func errorJSON(msg string) string {
	b, _ := jsonString("error", msg)
	return b
}

func jsonString(key, val string) (string, error) {
	return `{"file":"","diagnostics":[{"line":0,"col":0,"severity":"error","code":"WORKER_HARNESS","message":` + jsQuote(val) + `}]}`, nil
}

func jsQuote(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\':
			out = append(out, '\\', c)
		case '\n':
			out = append(out, '\\', 'n')
		case '\r':
			out = append(out, '\\', 'r')
		case '\t':
			out = append(out, '\\', 't')
		default:
			if c < 0x20 {
				const hex = "0123456789abcdef"
				out = append(out, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xf])
			} else {
				out = append(out, c)
			}
		}
	}
	out = append(out, '"')
	return string(out)
}
