package main

import (
	"flag"
	"fmt"
	"os"

	"void-slice/internal/lint"
	"void-slice/internal/report"
	"void-slice/internal/scan"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "lint" {
		fmt.Fprintln(os.Stderr, "usage: voidslice lint <file> [--json]")
		os.Exit(2)
	}

	lintFlags := flag.NewFlagSet("lint", flag.ExitOnError)
	jsonMode := lintFlags.Bool("json", false, "machine-readable JSON output")
	lintFlags.Parse(os.Args[2:]) //nolint

	if lintFlags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: voidslice lint <file> [--json]")
		os.Exit(2)
	}
	filename := lintFlags.Arg(0)

	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "voidslice: %v\n", err)
		os.Exit(2)
	}

	diags, err := lint.New().Lint(filename, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "voidslice: %v\n", err)
		os.Exit(2)
	}

	scanDiags := toScanDiags(diags)
	if *jsonMode {
		fmt.Print(report.RenderJSON(filename, src, scanDiags))
	} else {
		out := report.Render(filename, src, scanDiags, report.RenderOptions{})
		if out != "" {
			fmt.Print(out)
		}
	}

	for _, d := range diags {
		if d.Severity == lint.Error {
			os.Exit(1)
		}
	}
	os.Exit(0)
}

func toScanDiags(ds []lint.Diagnostic) []scan.Diagnostic {
	out := make([]scan.Diagnostic, len(ds))
	for i, d := range ds {
		out[i] = scan.Diagnostic{Code: scan.DiagnosticCode(d.Code), Span: d.Span, Message: d.Message}
	}
	return out
}
