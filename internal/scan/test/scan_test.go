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

var ingestedFiles map[string]string
var filesToIngest = []string{
	"game1/generated.decls.gamelogicmanager.ui.gamelogic.manager..gamelogicmanager.decl",
	// "game1/generated.decls.activeragdoll.models.characters.base.active_ragdoll.ar_hookmine_head_ceiling..activeragdoll.decl",
	"game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl",
	"game1/generated.decls.greatestmomentsmanager.greatestmoments.manager.manager..greatestmomentsmanager.decl",
	"game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities",
}

func TestMain(m *testing.M) {
	flag.Parse()

	testDir := filepath.FromSlash("../../../void-files/d2/")
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
	// file1 := ingestedFiles[filesToIngest[1]]

	require.NotEmpty(t, file0, "expected non-empty test file (file0)")

	tests := []tc{
		// {
		// 	name:      "sanity test",
		// 	bytes:     []byte("\n{\n\tedit = {\n\t\n\t}\n}"),
		// 	wantDiags: []scan.Diagnostic{}, // zero scanner errors should be produced
		// },
		// {
		// 	name:      "file0",
		// 	bytes:     []byte(file0),
		// 	wantDiags: []scan.Diagnostic{},
		// },
		{
			name:      "giant .entities file",
			bytes:     []byte(ingestedFiles[filesToIngest[3]]),
			wantDiags: []scan.Diagnostic{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("\n=== RUN %s\n", tt.name)
			// fmt.Printf("%s", tt.bytes)

			startTime := time.Now()
			result_tokens, result_diags := scan.Scan(tt.bytes)
			endTime := time.Now()

			assert.NotEmpty(t, result_tokens, "expected more than zero tokens")
			assert.ElementsMatch(t, tt.wantDiags, result_diags,
				"expected Diagnostics does not match actual Dignostics:\n"+
					"\tExpected\n%v\n\tActual\n%v\n", tt.wantDiags, result_diags)

			timeMillis := endTime.Sub(startTime).Milliseconds()
			fmt.Printf("%s\nconsumed %d bytes, produced %d tokens and %d diags\t(took %dms)", t.Name(), len(tt.bytes), len(result_tokens), len(result_diags), timeMillis)
		})
	}

}
