package lexinvariance_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"void-slice/internal/harness/lexinvariance"
	"void-slice/internal/lint"
	"void-slice/internal/scan"
)

// Synthetic fixture for the transform-preserves-tokens contract. Mixes
// tabs, multi-space indents, a comment line, a block comment, a quoted
// literal with embedded whitespace, multiple top-level constructs, and
// inter-token padding. Each transform's contract test re-uses it so the
// invariance assertion bites on a non-trivial input.
const contractFixture = "" +
	"// header line\n" +
	"Version 1\n" +
	"\n" +
	"component {\n" +
	"\tcpntTest myTest {\n" +
	"\t\tedit = {\n" +
	"\t\t\titem[0] = { m_val = \"a b\"; }\n" +
	"\t\t\titem[1] = { /* inline */ m_val = \"c\td\"; }\n" +
	"\t\t}\n" +
	"\t}\n" +
	"}\n"

func TestTransformReindentPreservesTokens(t *testing.T) {
	for v := 0; v < lexinvariance.VariantsPerKind(); v++ {
		v := v
		t.Run(name(v), func(t *testing.T) {
			assertContractHolds(t, lexinvariance.TransformReindent, v, contractFixture)
		})
	}
}

func TestTransformTabSpacePreservesTokens(t *testing.T) {
	for v := 0; v < lexinvariance.VariantsPerKind(); v++ {
		v := v
		t.Run(name(v), func(t *testing.T) {
			assertContractHolds(t, lexinvariance.TransformTabSpace, v, contractFixture)
		})
	}
}

func TestTransformInterTokenPaddingPreservesTokens(t *testing.T) {
	for v := 0; v < lexinvariance.VariantsPerKind(); v++ {
		v := v
		t.Run(name(v), func(t *testing.T) {
			assertContractHolds(t, lexinvariance.TransformInterTokenPadding, v, contractFixture)
		})
	}
}

func TestTransformBlankLineJitterPreservesTokens(t *testing.T) {
	for v := 0; v < lexinvariance.VariantsPerKind(); v++ {
		v := v
		t.Run(name(v), func(t *testing.T) {
			assertContractHolds(t, lexinvariance.TransformBlankLineJitter, v, contractFixture)
		})
	}
}

// TestTransformsSkipCommentInteriors pins that no transform edits bytes
// inside a COMMENT_LINE, COMMENT_BLOCK, or QUOTE_LITERAL span. The
// fixture deliberately seeds tabs and 4-space runs *inside* a block
// comment and a quoted literal to make the contract bite: if a transform
// rewrote them, the comment/quote tokens would differ byte-for-byte from
// the baseline.
func TestTransformsSkipCommentInteriors(t *testing.T) {
	const src = "" +
		"// a\tb    c\n" +
		"Version 1\n" +
		"component { /* x\ty    z */ a = \"u\tv    w\"; }\n"

	srcToks, _, _ := scan.Scan([]byte(src))
	var srcSpanBytes [][]byte
	for _, tk := range srcToks {
		switch tk.Kind {
		case scan.KindCommentLine, scan.KindCommentBlock, scan.KindQuoteLiteral:
			srcSpanBytes = append(srcSpanBytes, []byte(src[tk.Span.Start:tk.Span.End]))
		}
	}
	require.NotEmpty(t, srcSpanBytes, "fixture must contain at least one comment/quote token")

	for _, k := range lexinvariance.AllKinds() {
		for v := 0; v < lexinvariance.VariantsPerKind(); v++ {
			k, v := k, v
			t.Run(k.String()+"/"+name(v), func(t *testing.T) {
				out, ok := lexinvariance.Transform([]byte(src), k, v)
				if !ok {
					t.Skip("transform reported not-applicable")
				}
				outToks, _, _ := scan.Scan(out)
				var outSpanBytes [][]byte
				for _, tk := range outToks {
					switch tk.Kind {
					case scan.KindCommentLine, scan.KindCommentBlock, scan.KindQuoteLiteral:
						outSpanBytes = append(outSpanBytes, []byte(out[tk.Span.Start:tk.Span.End]))
					}
				}
				if !reflect.DeepEqual(srcSpanBytes, outSpanBytes) {
					t.Errorf("transform %s/v%d edited comment/quote interior bytes\nbefore: %q\nafter:  %q",
						k, v, srcSpanBytes, outSpanBytes)
				}
			})
		}
	}
}

// TestKnownCaseV1V2Diverges is the spike's anchor test: it pins that
// the V1 fixture from testdata/cascades/lexinvariance/ produces at least
// one Finding under TransformReindent or TransformBlankLineJitter. The
// 8-vs-2-diagnostic divergence motivated this whole ticket, so the
// harness landing without surfacing it would be a contract failure.
func TestKnownCaseV1V2Diverges(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "cascades", "lexinvariance", "whitespace-cascade.decl")
	src, err := os.ReadFile(path)
	require.NoError(t, err, "read V1 fixture")

	linter := lint.New()
	baseDiags, err := linter.Lint(path, src)
	require.NoError(t, err, "baseline lint must not error on the V1 fixture")
	baseScan := convertLintDiags(baseDiags)

	findings := lexinvariance.Scan(path, src, lexinvariance.Options{BaseDiags: baseScan})
	require.NotEmpty(t, findings, "the V1 fixture must produce at least one finding")

	wantKind := false
	for _, f := range findings {
		if f.Transform == lexinvariance.TransformReindent ||
			f.Transform == lexinvariance.TransformBlankLineJitter {
			wantKind = true
			break
		}
	}
	assert.True(t, wantKind,
		"V1 fixture must produce a finding under TransformReindent or TransformBlankLineJitter; got %d findings under other kinds",
		len(findings))
}

// assertContractHolds applies (kind, variant) to fixture and asserts that
// the post-transform lexical-token slice (whitespace and comments
// filtered out) equals the pre-transform slice. If the transform reports
// ok=false the test passes trivially — a not-applicable transform makes
// no claim about preservation.
func assertContractHolds(t *testing.T, kind lexinvariance.TransformKind, variantIdx int, fixture string) {
	t.Helper()
	src := []byte(fixture)
	before := lexinvariance.LexicalTokens(src)

	out, ok := lexinvariance.Transform(src, kind, variantIdx)
	if !ok {
		t.Skipf("transform %s/v%d reports not-applicable on the contract fixture", kind, variantIdx)
	}
	after := lexinvariance.LexicalTokens(out)

	if !reflect.DeepEqual(before, after) {
		t.Errorf("transform %s/v%d violated lexical-equivalence contract\nbefore tokens: %#v\nafter tokens:  %#v\nbefore bytes:  %q\nafter bytes:   %q",
			kind, variantIdx, before, after, string(src), string(out))
	}
}

func name(variantIdx int) string {
	return "v" + itoa(variantIdx)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func convertLintDiags(diags []lint.Diagnostic) []scan.Diagnostic {
	out := make([]scan.Diagnostic, len(diags))
	for i, d := range diags {
		out[i] = scan.Diagnostic{
			Code:    scan.DiagnosticCode(d.Code),
			Span:    d.Span,
			Message: d.Message,
		}
	}
	return out
}
