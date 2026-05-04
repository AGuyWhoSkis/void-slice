package parse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"void-slice/internal/parse"
	"void-slice/internal/scan"
)

// shapeName returns a stable label for a parse.Shape (test diagnostics only).
func shapeName(s parse.Shape) string {
	switch s {
	case parse.ShapeEntities:
		return "Entities"
	case parse.Shape1Curly:
		return "Shape1Curly"
	case parse.Shape2Animset:
		return "Shape2Animset"
	case parse.Shape3Material:
		return "Shape3Material"
	case parse.Shape4Renderprog:
		return "Shape4Renderprog"
	case parse.Shape5Md6def:
		return "Shape5Md6def"
	case parse.ShapeSidecarXML:
		return "ShapeSidecarXML"
	}
	return "?"
}

// goldenRel returns the absolute path to a corpus file under testdata/golden/.
func goldenRel(rel string) string {
	return filepath.Join("..", "..", "testdata", "golden", filepath.FromSlash(rel))
}

// scanFile reads a file and tokenizes it. Skips the test if the file is missing.
func scanFile(t *testing.T, path string) ([]byte, []scan.Token) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("corpus file unavailable: %v", err)
	}
	toks, _, _ := scan.Scan(src)
	return src, toks
}

// -------------------------
// Classify routing table — corpus representatives
// -------------------------

func TestClassify_Representatives(t *testing.T) {
	cases := []struct {
		rel  string
		want parse.Shape
	}{
		{"d2/game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities", parse.ShapeEntities},
		{"d2/game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl", parse.Shape1Curly},
		{"generated.decls.animset.models.weapons.timeshift_device.timeshift..animset.decl", parse.Shape2Animset},
		{"generated.decls.material.models.environment.voidhouse.rock_set.void_rock_01..material.decl", parse.Shape3Material},
		{"generated.decls.renderprog.arksssblur.decl", parse.Shape4Renderprog},
		{"doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl", parse.Shape5Md6def},
		{"generated.decls.animbasic.models.characters.dlc01.player.billie.additives_body..animbasic.decl.xml", parse.ShapeSidecarXML},
		// physicsmaterial-named file with Shape-1 content — content-sniff must beat filename.
		{"doto/game1/generated.decls.physicsmaterial.contactsystem.weapons.decl", parse.Shape1Curly},
	}

	for _, tc := range cases {
		t.Run(filepath.Base(tc.rel), func(t *testing.T) {
			path := goldenRel(tc.rel)
			src, toks := scanFile(t, path)
			got := parse.Classify(path, src, toks)
			assert.Equal(t, shapeName(tc.want), shapeName(got),
				"%s: expected %s, got %s", tc.rel, shapeName(tc.want), shapeName(got))
		})
	}
}

// -------------------------
// Classify edge cases — extension-only paths, empty input
// -------------------------

func TestClassify_ExtensionOnly(t *testing.T) {
	cases := []struct {
		path string
		want parse.Shape
	}{
		{"", parse.ShapeEntities},
		{"foo.entities", parse.ShapeEntities},
		{"foo.cfg", parse.ShapeEntities},
		{"foo.decl.xml", parse.ShapeSidecarXML},
		{"FOO.DECL.XML", parse.ShapeSidecarXML}, // case-insensitive
	}
	for _, tc := range cases {
		got := parse.Classify(tc.path, nil, nil)
		assert.Equal(t, shapeName(tc.want), shapeName(got), "path=%q", tc.path)
	}
}

// TestClassify_EmptyDecl — a .decl file with no tokens (or just comments)
// should fall through to Shape1Curly rather than panic.
func TestClassify_EmptyDecl(t *testing.T) {
	src := []byte("// just a comment\n")
	toks, _, _ := scan.Scan(src)
	got := parse.Classify("empty.decl", src, toks)
	assert.Equal(t, parse.Shape1Curly, got)
}

// -------------------------
// Walk: stub shapes produce zero events and zero diagnostics
// -------------------------

// zeroEventsCases pairs each non-Shape-1 representative with its expected
// (zero) event count. Stub walkers don't traverse the body, so anything past
// the dispatch decision should produce nothing.
var zeroEventsCases = []struct {
	rel  string
	name string
}{
	{"generated.decls.animset.models.weapons.timeshift_device.timeshift..animset.decl", "shape2"},
	{"generated.decls.material.models.environment.voidhouse.rock_set.void_rock_01..material.decl", "shape3"},
	{"generated.decls.renderprog.arksssblur.decl", "shape4"},
	{"doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl", "shape5"},
	{"generated.decls.animbasic.models.characters.dlc01.player.billie.additives_body..animbasic.decl.xml", "sidecar"},
}

func TestWalk_StubsEmitNoEventsOrDiags(t *testing.T) {
	for _, tc := range zeroEventsCases {
		t.Run(tc.name, func(t *testing.T) {
			path := goldenRel(tc.rel)
			src, toks := scanFile(t, path)
			rec := newRecorder(src)
			diags := parse.Walk(path, src, toks, rec)
			assert.Empty(t, diags, "%s: stub walker must produce zero diagnostics, got %d", tc.name, len(diags))
			assert.Empty(t, rec.events, "%s: stub walker must produce zero events, got %d", tc.name, len(rec.events))
		})
	}
}

// -------------------------
// Walk: Shape 1 path matches WalkEntities behaviour
// -------------------------

func TestWalk_Shape1MatchesWalkEntities(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		m_x = 1;
	}
}`)
	toks, scanDiags, _ := scan.Scan(src)
	require.Empty(t, scanDiags)

	walkRec := newRecorder(src)
	walkDiags := parse.Walk("test.entities", src, toks, walkRec)

	directRec := newRecorder(src)
	directDiags := parse.WalkEntities(src, toks, directRec)

	assert.Equal(t, eventKinds(directRec), eventKinds(walkRec),
		"Walk through ShapeEntities path must produce the same event kinds as WalkEntities")
	assert.Len(t, walkDiags, len(directDiags))
}

// TestWalk_Shape1DeclSniffsCorrectly — a `.decl` file that starts with
// `Version` (not a sniff keyword) must route to the Shape-1 walker.
func TestWalk_Shape1DeclSniffsCorrectly(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		m_x = 1;
	}
}`)
	toks, _, _ := scan.Scan(src)
	rec := newRecorder(src)
	diags := parse.Walk("test.decl", src, toks, rec)
	require.Empty(t, diags)
	assert.NotEmpty(t, filterKind(rec.events, "ComponentDecl"),
		"Shape-1 .decl must produce component events")
}
