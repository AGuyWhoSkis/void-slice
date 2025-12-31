package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"void-slice/internal/core"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRun_CanonicalEntry_CopiesTransitiveAcrossBuckets_CollectsUnresolved(t *testing.T) {
	tmp := t.TempDir()

	exportRoot := filepath.Join(tmp, "Export")

	// Buckets
	game1 := filepath.Join(exportRoot, "game1")
	game2 := filepath.Join(exportRoot, "game2")

	// Canonical names (appear in ResourceInfo@name)
	A := "models/a/a.asset"
	Missing := "sounds/ui/click.wav"

	// Exported filenames (flattened)
	// A lives in game1, B and C live in game2 (cross-bucket).
	writeFile(t,
		filepath.Join(game1, "A.decl.xml"),
		`<ResourceInfo name="models/a/a.asset" classname="X" isCompressed="true" />`,
	)
	writeFile(t,
		filepath.Join(game1, "A.decl"),
		`ref components/b/b.rulehandler
ref sounds/ui/click.wav
`,
	)

	writeFile(t,
		filepath.Join(game2, "B.decl.xml"),
		`<ResourceInfo name="components/b/b.rulehandler" classname="Y" isCompressed="true" />`,
	)
	writeFile(t,
		filepath.Join(game2, "B.decl"),
		`ref models/c/c.png`,
	)

	writeFile(t,
		filepath.Join(game2, "C.decl.xml"),
		`<ResourceInfo name="models/c/c.png" classname="Z" isCompressed="true" />`,
	)
	writeFile(t,
		filepath.Join(game2, "C.decl"),
		`// no refs`,
	)

	outDir := filepath.Join(tmp, "out")

	result, err := core.Run(core.Options{
		ExportRoot: exportRoot,
		Entry:      A,
		OutDir:     outDir,
		MaxDepth:   10,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Root + B + C should be copied, across their buckets
	mustExist := []string{
		filepath.Join(outDir, "game1", "A.decl"),
		filepath.Join(outDir, "game1", "A.decl.xml"),

		filepath.Join(outDir, "game2", "B.decl"),
		filepath.Join(outDir, "game2", "B.decl.xml"),

		filepath.Join(outDir, "game2", "C.decl"),
		filepath.Join(outDir, "game2", "C.decl.xml"),
	}
	for _, p := range mustExist {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file to exist: %s (err=%v)", p, err)
		}
	}

	// Missing should be recorded unresolved
	foundMissing := false
	for _, u := range result.Unresolved {
		if u == Missing {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatalf("expected unresolved to include %q; got: %#v", Missing, result.Unresolved)
	}
}

func TestRun_DepthLimitWarns(t *testing.T) {
	tmp := t.TempDir()
	exportRoot := filepath.Join(tmp, "Export")
	game1 := filepath.Join(exportRoot, "game1")

	// Chain A->B->C... length 12 to exceed max depth 10 (depending on definition).
	// We'll create 12 nodes; each references the next.
	names := make([]string, 12)
	for i := range names {
		names[i] = "models/chain/node" + string(rune('A'+i))
	}

	for i := 0; i < 12; i++ {
		declName := string(rune('A'+i)) + ".decl"
		writeFile(t,
			filepath.Join(game1, declName+".xml"),
			`<ResourceInfo name="`+names[i]+`" classname="X" isCompressed="true" />`,
		)
		content := `// end`
		if i < 11 {
			content = `ref ` + names[i+1]
		}
		writeFile(t, filepath.Join(game1, declName), content)
	}

	outDir := filepath.Join(tmp, "out")
	result, err := core.Run(core.Options{
		ExportRoot: exportRoot,
		Entry:      names[0],
		OutDir:     outDir,
		MaxDepth:   10,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// We expect at least one warning about depth limit exceeded.
	hasWarn := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "depth limit exceeded") {
			hasWarn = true
			break
		}
	}
	if !hasWarn {
		t.Fatalf("expected a depth warning; warnings=%#v", result.Warnings)
	}
}
