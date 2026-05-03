package lint

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"void-slice/internal/parse"
	"void-slice/internal/scan"
	"void-slice/internal/validate"
)

type Severity int

const (
	Error   Severity = iota
	Warning
)

type Diagnostic struct {
	Severity Severity
	Code     string
	Span     scan.Span
	Message  string
}

type Linter interface {
	Lint(filename string, src []byte) ([]Diagnostic, error)
}

// Options configures a Linter. The zero value is "no limits" — the right
// default for callers running on end users' machines (CLI, LSP / VS Code
// extension, local dev server). The Worker entry point sets MaxDiagnostics
// to bound memory usage inside its 128 MB isolate.
type Options struct {
	// MaxDiagnostics caps the total number of diagnostics returned by Lint.
	// 0 = unlimited. When the cap is reached, the final entry is replaced
	// with a PARSE_DIAGNOSTICS_TRUNCATED sentinel.
	MaxDiagnostics int
}

func New() Linter {
	return &linter{}
}

func NewWithOptions(opts Options) Linter {
	return &linter{opts: opts}
}

type linter struct {
	opts Options
}

func (l *linter) Lint(filename string, src []byte) ([]Diagnostic, error) {
	var out []Diagnostic

	action, warn := classifyFile(filename)
	if action == actionBinary {
		return []Diagnostic{{
			Severity: Error,
			Code:     "LINT_BINARY_FILE",
			Message:  "binary map file — cannot lint",
		}}, nil
	}
	if warn != nil {
		out = append(out, *warn)
	}

	n := 512
	if len(src) < n {
		n = len(src)
	}
	if action != actionSkipBinarySniff && isBinary(src[:n]) {
		return []Diagnostic{{
			Severity: Error,
			Code:     "LINT_BINARY_FILE",
			Message:  "binary map file — cannot lint",
		}}, nil
	}

	toks, scanDiags, _ := scan.Scan(src)
	validateDiags := validate.ValidateEntities(src, toks, parse.Opts{MaxDiagnostics: l.opts.MaxDiagnostics})

	for _, d := range scanDiags {
		out = append(out, convert(d))
	}
	for _, d := range validateDiags {
		out = append(out, convert(d))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Span.Start < out[j].Span.Start
	})
	if l.opts.MaxDiagnostics > 0 && len(out) > l.opts.MaxDiagnostics {
		out = out[:l.opts.MaxDiagnostics]
		out[len(out)-1] = Diagnostic{
			Severity: Error,
			Code:     string(parse.Codes.DIAGNOSTICS_TRUNCATED),
			Span:     scan.NewSpan(len(src), len(src)),
			Message:  fmt.Sprintf("diagnostic limit (%d) reached; further diagnostics omitted", l.opts.MaxDiagnostics),
		}
	}
	return out, nil
}

type fileAction int

const (
	actionLint fileAction = iota
	actionBinary
	actionSkipBinarySniff // known text types — skip the sniff, just lint
)

// Convention: the `default` branch handles "unknown / no filetype context"
// (e.g. the playground, where the file's true extension can't be known) and
// MUST NOT emit filetype-derived warnings. Warnings whose meaning depends on
// the file's extension belong on a known-extension branch, not the default.
// The playground relies on this to suppress filetype-sensitive noise by
// passing an extensionless filename.
func classifyFile(filename string) (fileAction, *Diagnostic) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".decl", ".entitydef":
		return actionSkipBinarySniff, nil
	case ".entities", ".cfg":
		w := &Diagnostic{
			Severity: Warning,
			Code:     "LINT_VE_INCONSISTENCY",
			Message:  "editing via Void Explorer is inconsistent — results may not reflect in-game",
		}
		return actionSkipBinarySniff, w
	case ".tome", ".bwm", ".navmesh", ".mapresources",
		".soundpropa", ".bnavmesh", ".bphysworld", ".maprscreusechunk0":
		return actionBinary, nil
	default:
		return actionLint, nil
	}
}

func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0x00 {
			return true
		}
	}
	return false
}

func convert(d scan.Diagnostic) Diagnostic {
	sev := Error
	if strings.HasPrefix(string(d.Code), "VALIDATE_") {
		sev = Warning
	}
	return Diagnostic{
		Severity: sev,
		Code:     string(d.Code),
		Span:     d.Span,
		Message:  d.Message,
	}
}
