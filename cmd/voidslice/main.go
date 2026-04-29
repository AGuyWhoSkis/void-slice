package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"void-slice/internal/lint"
	"void-slice/internal/lsp"
	"void-slice/internal/report"
	"void-slice/internal/scan"
	"void-slice/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "lint":
		os.Exit(runLint(os.Args[2:]))
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "lsp":
		os.Exit(runLSP())
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  voidslice lint <file> [--json]")
	fmt.Fprintln(os.Stderr, "  voidslice serve [--port 8080]")
	fmt.Fprintln(os.Stderr, "  voidslice lsp")
}

func runLint(args []string) int {
	lintFlags := flag.NewFlagSet("lint", flag.ExitOnError)
	jsonMode := lintFlags.Bool("json", false, "machine-readable JSON output")
	lintFlags.Parse(args) //nolint

	if lintFlags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: voidslice lint <file> [--json]")
		return 2
	}
	filename := lintFlags.Arg(0)

	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "voidslice: %v\n", err)
		return 2
	}

	diags, err := lint.New().Lint(filename, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "voidslice: %v\n", err)
		return 2
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
			return 1
		}
	}
	return 0
}

func runServe(args []string) int {
	serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)
	port := serveFlags.Int("port", 8080, "TCP port to listen on")
	serveFlags.Parse(args) //nolint

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	cfg := server.Config{
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
		Logger:        logger,
	}

	addr := ":" + strconv.Itoa(*port)
	logger.Info("server starting", slog.String("addr", addr), slog.String("allowed_origin", originOrDefault(cfg.AllowedOrigin)))
	if err := server.ListenAndServe(addr, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "voidslice: %v\n", err)
		return 1
	}
	return 0
}

func originOrDefault(o string) string {
	if o == "" {
		return "*"
	}
	return o
}

func runLSP() int {
	if err := lsp.New().Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "voidslice lsp: %v\n", err)
		return 1
	}
	return 0
}

func toScanDiags(ds []lint.Diagnostic) []scan.Diagnostic {
	out := make([]scan.Diagnostic, len(ds))
	for i, d := range ds {
		out[i] = scan.Diagnostic{Code: scan.DiagnosticCode(d.Code), Span: d.Span, Message: d.Message}
	}
	return out
}
