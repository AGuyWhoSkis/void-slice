package scan_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
	"void-slice/internal/scan"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A tool for double checking byte index offsets: https://large-type.com
// 	Use this JS console command to switch it to 0-index
//		document.querySelector(".text").style.counterReset = "num-chars -1";

var testDir = filepath.FromSlash("../../testdata/golden/")

var goldenFiles map[string][]byte
var goldenFileNames = []string{
	"d2/game1/generated.decls.gamelogicmanager.ui.gamelogic.manager..gamelogicmanager.decl",
	"d2/game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl",
	"d2/game1/generated.decls.greatestmomentsmanager.greatestmoments.manager.manager..greatestmomentsmanager.decl",
	"d2/game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities",
	"doto/game1/generated.decls.physicsmaterial.contactsystem.weapons.decl",
	"doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl",
}

func TestMain(m *testing.M) {
	flag.Parse()

	goldenFiles = make(map[string][]byte)

	for _, filename := range goldenFileNames {
		filePath := filepath.Join(testDir, filename)
		b, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "test setup failed: read %q: %v\n", filePath, err)
			os.Exit(1)
		}
		goldenFiles[filename] = b
	}

	os.Exit(m.Run())
}

func TestScanner(t *testing.T) {
	type tc struct {
		name      string
		bytes     []byte
		wantDiags []scan.Diagnostic
		wantToks  []scan.Token
	}

	tests := []tc{
		{
			name:      "valid snippet",
			bytes:     []byte("\n{\n\tedit = {\n\t\n\t}\n}"),
			wantDiags: []scan.Diagnostic{},
		},
		{
			name:  "unterminated-quote",
			bytes: []byte(`{ edit = { m_max = "missing-right-quote ; } }`), // len=45
			wantDiags: []scan.Diagnostic{
				{Code: scan.Codes.SCAN, Span: scan.NewSpan(19, 45), Message: "unterminated quote"},
			},
			wantToks: []scan.Token{
				{Kind: 0, Span: scan.NewSpan(0, 1)},   // spans `{`
				{Kind: 1, Span: scan.NewSpan(2, 6)},   // spans `edit`
				{Kind: 0, Span: scan.NewSpan(7, 8)},   // spans `=`
				{Kind: 0, Span: scan.NewSpan(9, 10)},  // spans `{`
				{Kind: 1, Span: scan.NewSpan(11, 16)}, // spans `m_max`
				{Kind: 0, Span: scan.NewSpan(17, 18)}, // spans `=`
				{Kind: 2, Span: scan.NewSpan(19, 45)}, // spans `"missing-right-hand-quot ; } }`
			},
		},
		{
			name:  "unteriminated-number",
			bytes: []byte("{ edit = -123.0f"), // len=16
			wantDiags: []scan.Diagnostic{
				{Code: scan.Codes.SCAN, Span: scan.NewSpan(9, 16), Message: "unterminated number literal"},
			},
			wantToks: []scan.Token{
				{Kind: 0, Span: scan.NewSpan(0, 1)},
				{Kind: 1, Span: scan.NewSpan(2, 6)},
				{Kind: 0, Span: scan.NewSpan(7, 8)},
				{Kind: 3, Span: scan.NewSpan(9, 16)},
			},
		},
		{
			name:  "unknown byte",
			bytes: []byte("{ \xff }"),
			wantDiags: []scan.Diagnostic{
				{Code: scan.Codes.SCAN, Span: scan.NewSpan(2, 3), Message: "unknown byte 11111111"},
			},
			wantToks: []scan.Token{
				{Kind: scan.TokenKind.SYMBOL, Span: scan.NewSpan(0, 1)},
				{Kind: scan.TokenKind.SYMBOL, Span: scan.NewSpan(4, 5)},
			},
		},
		{
			// `,` `(` `)` `:` are SYMBOL: shape-1 tuples (`color = ( 1, 1, 1 );`),
			// shape-2 scoped names (`Foo::Bar`), shape-3 function-call values.
			name:      "comma paren colon symbols",
			bytes:     []byte("(1,2):"),
			wantDiags: []scan.Diagnostic{},
			wantToks: []scan.Token{
				{Kind: scan.TokenKind.SYMBOL, Span: scan.NewSpan(0, 1)},         // (
				{Kind: scan.TokenKind.NUMBER_LITERAL, Span: scan.NewSpan(1, 2)}, // 1
				{Kind: scan.TokenKind.SYMBOL, Span: scan.NewSpan(2, 3)},         // ,
				{Kind: scan.TokenKind.NUMBER_LITERAL, Span: scan.NewSpan(3, 4)}, // 2
				{Kind: scan.TokenKind.SYMBOL, Span: scan.NewSpan(4, 5)},         // )
				{Kind: scan.TokenKind.SYMBOL, Span: scan.NewSpan(5, 6)},         // :
			},
		},
		{
			name:  "unterminated-block-comment",
			bytes: []byte("/* foo"), // len=6
			wantDiags: []scan.Diagnostic{
				{Code: scan.Codes.SCAN, Span: scan.NewSpan(0, 6), Message: "unterminated block comment"},
			},
			wantToks: []scan.Token{
				{Kind: scan.TokenKind.COMMENT_BLOCK, Span: scan.NewSpan(0, 6)},
			},
		},
		{
			name:      "line-comment-eof-no-newline",
			bytes:     []byte("// foo"), // len=6
			wantDiags: []scan.Diagnostic{},
			wantToks: []scan.Token{
				{Kind: scan.TokenKind.COMMENT_LINE, Span: scan.NewSpan(0, 6)},
			},
		},
		{
			// CRLF: COMMENT_LINE span must not include the trailing '\r'.
			// Source: "// foo\r\nbar"
			//   bytes:  /  /  ' '  f  o  o  \r  \n  b  a  r
			//   offset: 0  1   2   3  4  5   6   7  8  9 10
			name:      "CRLF line comment trims trailing CR",
			bytes:     []byte("// foo\r\nbar"),
			wantDiags: []scan.Diagnostic{},
			wantToks: []scan.Token{
				{Kind: scan.TokenKind.COMMENT_LINE, Span: scan.NewSpan(0, 6)}, // "// foo" — no \r
				{Kind: scan.TokenKind.IDENTIFIER, Span: scan.NewSpan(8, 11)},  // "bar"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("\n=== RUN %s\n", tt.name)

			startTime := time.Now()
			result_tokens, result_diags, _ := scan.Scan(tt.bytes)
			endTime := time.Now()

			assert.NotEmpty(t, result_tokens, "expected more than zero tokens")
			assert.ElementsMatch(t, tt.wantDiags, result_diags,
				"expected Diagnostics does not match actual Dignostics:\n"+
					"\tExpected\n%v\n\tActual\n%v\n", tt.wantDiags, result_diags)

			if tt.wantToks != nil {
				for i, wantToken := range tt.wantToks {
					require.NotNil(t, wantToken, "how the heck is this nil anyway (wantToken %d)", i)
					require.NotNil(t, wantToken.Kind, "wantToken cannot have nil Kind")
				}
				assert.ElementsMatchf(t, tt.wantToks, result_tokens, "expected tokens to match: \n\tExpected\n%v\n\tActual\n%v", tt.wantToks, result_tokens)
			}

			timeMillis := endTime.Sub(startTime).Milliseconds()
			fmt.Printf("%s\nconsumed %d bytes, produced %d tokens and %d diags\t(took %dms)", t.Name(), len(tt.bytes), len(result_tokens), len(result_diags), timeMillis)
		})
	}

}

type spotCheck struct {
	offset   int       // byte offset in source
	wantKind scan.Kind // expected token kind
	wantText string    // expected lexeme (if empty, skip text check)
}

// TestScannerGoldenFiles validates the scanner against large, syntactically valid files
// Strategy:
//  1. All golden files should produce zero diagnostics (they're syntactically valid)
//  2. Verify token integrity: spans are sequential, non-overlapping, cover full input
//  3. Spot-check tokens at specific byte offsets to ensure correctness
//  4. Performance benchmarking for large files
func TestScannerGoldenFiles(t *testing.T) {
	file0 := goldenFiles[goldenFileNames[0]]
	file1 := goldenFiles[goldenFileNames[1]]
	file2 := goldenFiles[goldenFileNames[2]]
	file3 := goldenFiles[goldenFileNames[3]]
	file4 := goldenFiles[goldenFileNames[4]]
	file5 := goldenFiles[goldenFileNames[5]]

	require.NotEmpty(t, file0, "expected non-empty test file (file0)")
	require.NotEmpty(t, file1, "expected non-empty test file (file1)")
	require.NotEmpty(t, file2, "expected non-empty test file (file2)")
	require.NotEmpty(t, file3, "expected non-empty test file (file3)")
	require.NotEmpty(t, file4, "expected non-empty test file (file4)")
	require.NotEmpty(t, file5, "expected non-empty test file (file5)")

	type tc struct {
		name       string
		bytes      []byte
		spotChecks []spotCheck // optional: validate specific tokens
	}

	tcs := []tc{
		{
			name:  ".entities (largest file)",
			bytes: []byte(file3),
			spotChecks: []spotCheck{
				{offset: 0, wantKind: scan.TokenKind.IDENTIFIER, wantText: "Version"},
				{offset: 8, wantKind: scan.TokenKind.NUMBER_LITERAL, wantText: "6"},
				{offset: 10, wantKind: scan.TokenKind.IDENTIFIER, wantText: "component"},
				{offset: 20, wantKind: scan.TokenKind.SYMBOL, wantText: "{"},
				{offset: 25980, wantKind: scan.TokenKind.NUMBER_LITERAL, wantText: "20150324"},
				// Add more spot checks for different sections of the file
			},
		},
		{
			name:  ".gamelogicmanager",
			bytes: []byte(file0),
			spotChecks: []spotCheck{
				{offset: 0, wantKind: scan.TokenKind.SYMBOL, wantText: "{"},
				{offset: 4, wantKind: scan.TokenKind.IDENTIFIER, wantText: "edit"},
			},
		},
		{
			name:  ".cpntplayerfxmanager",
			bytes: []byte(file1),
			spotChecks: []spotCheck{
				{offset: 0, wantKind: scan.TokenKind.SYMBOL, wantText: "{"},
				{offset: 4, wantKind: scan.TokenKind.IDENTIFIER, wantText: "edit"},
			},
		},
		{
			name:  ".greatestmomentsmanager",
			bytes: []byte(file2),
			spotChecks: []spotCheck{
				{offset: 0, wantKind: scan.TokenKind.SYMBOL, wantText: "{"},
				{offset: 4, wantKind: scan.TokenKind.IDENTIFIER, wantText: "edit"},
				{offset: 43, wantKind: scan.TokenKind.IDENTIFIER, wantText: "num"},
				{offset: 49, wantKind: scan.TokenKind.NUMBER_LITERAL, wantText: "10"},
			},
		},
		{
			name:  ".physicsmaterial (doto)",
			bytes: []byte(file4),
			spotChecks: []spotCheck{
				{offset: 0, wantKind: scan.TokenKind.SYMBOL, wantText: "{"},
				{offset: 4, wantKind: scan.TokenKind.IDENTIFIER, wantText: "inherit"},
				{offset: 100, wantKind: scan.TokenKind.NUMBER_LITERAL, wantText: "20150324"},
			},
		},
		{
			name:  ".md6def docker head (doto)",
			bytes: []byte(file5),
			spotChecks: []spotCheck{
				{offset: 0, wantKind: scan.TokenKind.SYMBOL, wantText: "{"},
				{offset: 4, wantKind: scan.TokenKind.IDENTIFIER, wantText: "init"},
				{offset: 105, wantKind: scan.TokenKind.IDENTIFIER, wantText: "lod"},
				{offset: 109, wantKind: scan.TokenKind.NUMBER_LITERAL, wantText: "0"},
			},
		},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("\n=== RUN %s\n", tt.name)

			startTime := time.Now()
			result_tokens, result_diags, _ := scan.Scan(tt.bytes)
			endTime := time.Now()

			// 1. Validate zero diagnostics (file is syntactically valid)
			assert.Empty(t, result_diags,
				"expected zero diagnostics for valid file but got %d:\n%v",
				len(result_diags), result_diags)

			// 2. Validate we got tokens
			require.NotEmpty(t, result_tokens, "expected scanner to produce tokens")

			// 3. Validate token integrity: sequential, non-overlapping spans
			validateTokenIntegrity(t, tt.bytes, result_tokens)

			// 4. Spot-check specific tokens
			if len(tt.spotChecks) > 0 {
				validateSpotChecks(t, tt.bytes, result_tokens, tt.spotChecks)
			}

			// 5. Report metrics
			timeMillis := endTime.Sub(startTime).Milliseconds()
			tokensPerKB := float64(len(result_tokens)) / (float64(len(tt.bytes)) / 1024.0)
			fmt.Printf("%s\n  consumed %d bytes (%.1f KB)\n  produced %d tokens (%.1f tokens/KB)\n  took %dms (%.2f MB/s)\n",
				t.Name(),
				len(tt.bytes),
				float64(len(tt.bytes))/1024.0,
				len(result_tokens),
				tokensPerKB,
				timeMillis,
				float64(len(tt.bytes))/1024.0/1024.0/(float64(timeMillis)/1000.0))
		})
	}
}

// validateTokenIntegrity ensures tokens are well-formed:
//   - spans are sequential (each token starts where last ended or after whitespace)
//   - spans are valid (Start < End, both within bounds)
//   - no overlapping spans
func validateTokenIntegrity(t *testing.T, src []byte, tokens []scan.Token) {
	n := len(src)

	for i, tok := range tokens {
		// Validate span bounds
		require.GreaterOrEqual(t, tok.Span.Start, 0,
			"token %d: invalid negative start offset", i)
		require.LessOrEqual(t, tok.Span.End, n,
			"token %d: end offset exceeds source length", i)
		require.Less(t, tok.Span.Start, tok.Span.End,
			"token %d: invalid span (start >= end)", i)

		// Validate sequential ordering (allowing for skipped whitespace)
		if i > 0 {
			prevEnd := tokens[i-1].Span.End
			require.GreaterOrEqual(t, tok.Span.Start, prevEnd,
				"token %d: span overlaps with previous token", i)
		}
	}
}

// validateSpotChecks verifies specific tokens at given byte offsets
func validateSpotChecks(t *testing.T, src []byte, tokens []scan.Token, checks []spotCheck) {
	fmt.Fprintf(os.Stderr, "\n%d tokens %d spot checks", len(tokens), len(checks))

	for _, check := range checks {
		// Find token containing or starting at this offset
		tok := findTokenAtOffset(tokens, check.offset)
		require.NotNil(t, tok,
			"no token found at offset %d", check.offset)

		require.NotNil(t, tok,
			"no token found at offset %d", check.offset)

		// Validate token kind
		assert.Equal(t, check.wantKind, tok.Kind,
			"offset %d: expected kind %v, got %v",
			check.offset, check.wantKind, tok.Kind)

		// Validate lexeme if specified
		if check.wantText != "" {
			actualText := string(src[tok.Span.Start:tok.Span.End])
			assert.Equal(t, check.wantText, actualText,
				"offset %d: expected lexeme %q, got %q",
				check.offset, check.wantText, actualText)
		}
	}
}

// findTokenAtOffset returns the first token that starts at or contains the given offset
func findTokenAtOffset(tokens []scan.Token, offset int) *scan.Token {
	for i := range tokens {
		if tokens[i].Span.Start <= offset && offset < tokens[i].Span.End {
			return &tokens[i]
		}
		if tokens[i].Span.Start > offset {
			break
		}
	}
	return nil
}

func Test_findTokensAtOffsets(t *testing.T) {
	tokens := []scan.Token{
		{Kind: 1, Span: scan.NewSpan(0, 1)},
		{Kind: 2, Span: scan.NewSpan(1, 3)},
		{Kind: 3, Span: scan.NewSpan(5, 6)},
	}

	got := scan.FindTokensAtOffsets(tokens, 0, 2, 4, 5, 10, -1, 2)

	require.Len(t, got, 7)

	require.NotNil(t, got[0])
	assert.Equal(t, scan.Kind(1), got[0].Kind)
	assert.Equal(t, scan.NewSpan(0, 1), got[0].Span)

	require.NotNil(t, got[1])
	assert.Equal(t, scan.Kind(2), got[1].Kind)
	assert.Equal(t, scan.NewSpan(1, 3), got[1].Span)

	assert.Nil(t, got[2]) // 4 is in a gap (no token contains it)

	require.NotNil(t, got[3])
	assert.Equal(t, scan.Kind(3), got[3].Kind)
	assert.Equal(t, scan.NewSpan(5, 6), got[3].Span)

	assert.Nil(t, got[4]) // beyond last token
	assert.Nil(t, got[5]) // negative offset

	// Duplicate offset: should map to the same token span as earlier offset 2.
	require.NotNil(t, got[6])
	assert.Equal(t, scan.Kind(2), got[6].Kind)
	assert.Equal(t, scan.NewSpan(1, 3), got[6].Span)
}

func Test_findTokensAtOffsets_panicsOnNoOffsets(t *testing.T) {
	tokens := []scan.Token{
		{Kind: 1, Span: scan.NewSpan(0, 1)},
	}

	require.Panics(t, func() {
		_ = scan.FindTokensAtOffsets(tokens)
	})
}
