package scan_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"void-slice/internal/scan"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ingestedFiles map[string]string
var filesToIngest = []string{
	"generated.decls.activeragdoll.models.characters.base.active_ragdoll.ar_hookmine_head_ceiling..activeragdoll.decl",
}

func TestMain(m *testing.M) {
	flag.Parse()

	testDir := filepath.FromSlash("../../../void-files/big-Export/game1")
	ingestedFiles = make(map[string]string)

	for _, filename := range filesToIngest {
		fp := filepath.Join(testDir, filename)
		b, err := os.ReadFile(fp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "test setup failed: read %q: %v\n", fp, err)
			os.Exit(1)
		}
		ingestedFiles[filename] = string(b)
	}

	os.Exit(m.Run())
}

func Test1(t *testing.T) {

}

func TestScanner(t *testing.T) {
	type tc struct {
		name      string
		bytes     []byte
		wantDiags []scan.Diagnostic
	}

	file0 := ingestedFiles[filesToIngest[0]]

	require.NotEmpty(t, file0, "expected non-empty test file (file0)")

	tests := []tc{
		{
			name:      "sanity test",
			bytes:     []byte("\n{\n\tedit = {\n\t\n\t}\n}"),
			wantDiags: []scan.Diagnostic{}, // zero scanner errors should be produced
		},
		{
			name:      "file0",
			bytes:     []byte(file0),
			wantDiags: []scan.Diagnostic{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("\n=== RUN %s\n", tt.name)
			fmt.Printf("%s", tt.bytes)
			result_tokens, result_diags := scan.Scan(tt.bytes)

			assert.NotEmpty(t, result_tokens, "expected more than zero tokens")

			for _, token := range result_tokens {
				fmt.Printf("\n\t%v", token)
			}

			for _, diag := range result_diags {
				fmt.Printf("\n\t%v", diag)
				// fmt.Printf("\n\t%s", diag.Code, diag.Severity, diag.Span)
			}
		})
	}

}
