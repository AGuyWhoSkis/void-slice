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

var testDir = filepath.FromSlash("../../../void-files/d2/")

var ingestedFiles map[string]string
var filesToIngest = []string{
	"game1/generated.decls.gamelogicmanager.ui.gamelogic.manager..gamelogicmanager.decl",
	"game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl",
	"game1/generated.decls.greatestmomentsmanager.greatestmoments.manager.manager..greatestmomentsmanager.decl",
	"game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities",
}

func TestMain(m *testing.M) {
	flag.Parse()

	ingestedFiles = make(map[string]string)

	for _, filename := range filesToIngest {
		filePath := filepath.Join(testDir, filename)
		b, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "test setup failed: read %q: %v\n", filePath, err)
			os.Exit(1)
		}
		ingestedFiles[filename] = string(b)
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

	file0 := ingestedFiles[filesToIngest[0]]
	file1 := ingestedFiles[filesToIngest[1]]
	file2 := ingestedFiles[filesToIngest[2]]

	require.NotEmpty(t, file0, "expected non-empty test file (file0)")
	require.NotEmpty(t, file1, "expected non-empty test file (file1)")
	require.NotEmpty(t, file2, "expected non-empty test file (file2)")

	tests := []tc{
		{
			name:      "valid snippet",
			bytes:     []byte("\n{\n\tedit = {\n\t\n\t}\n}"),
			wantDiags: []scan.Diagnostic{},
		},
		{
			name:  "unterminated-quote",
			bytes: []byte(`{ edit = { m_max = "missing-right-hand-quote ; } }`), // len=50; final byte index=49
			wantDiags: []scan.Diagnostic{
				{
					Code:     scan.Codes.SCAN,
					Severity: scan.Severities.ERROR,
					Span:     scan.NewSpan(19, 50),
					Message:  "unterminated quote",
				},
			},
		},
		{
			name:      "giant .entities file",
			bytes:     []byte(ingestedFiles[filesToIngest[3]]),
			wantDiags: []scan.Diagnostic{},
			wantToks: []scan.Token{
				{
					Kind: scan.TokenKind.QUOTE_LITERAL,
					Span: scan.NewSpan(1, 2),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("\n=== RUN %s\n", tt.name)

			startTime := time.Now()
			result_tokens, result_diags := scan.Scan(tt.bytes)
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
