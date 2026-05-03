package lint

import (
	"path/filepath"
	"sort"
	"strings"

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

func New() Linter {
	return &linter{}
}

type linter struct{}

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
	validateDiags := validate.ValidateEntities(src, toks)

	for _, d := range scanDiags {
		out = append(out, convert(d))
	}
	for _, d := range validateDiags {
		out = append(out, convert(d))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Span.Start < out[j].Span.Start
	})
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
