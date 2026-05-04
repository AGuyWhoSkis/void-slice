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

// Per-shape golden coverage for M4.4–M4.6: parse a representative corpus file
// for each in-scope shape and assert zero parse diagnostics. Validate-layer
// warnings (array-count, missing-num) are out of scope here — see M4.8.
func TestWalk_ShapeRepresentatives_ZeroParseDiags(t *testing.T) {
	cases := []struct {
		name        string
		rel         string
		wantBegin   bool // expect at least one ObjectBegin event
		wantTypBlk  bool // expect at least one TypedBlock event
		wantAssign  bool // expect at least one Assignment event
	}{
		// Shape 1 — curly inherit/edit. Sub-types covered: cpntplayerfxmanager
		// (basic edit), entitydef (inherit + tuple `color = ( 1, 1, 1, 1 );` +
		// editorVars bare block), arktree (deeply nested), kiscule, fx,
		// mapinfo, particledt, midnightscene, cpntplayercontroller.
		{name: "shape1/cpntplayerfxmanager",
			rel:        "d2/game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl",
			wantBegin:  true, wantAssign: true},
		{name: "shape1/entitydef-tuple-and-editorVars",
			rel:        "generated.decls.entitydef.models.characters.def.story.delilah_copperspoon..def.decl",
			wantBegin:  true, wantTypBlk: true, wantAssign: true},
		{name: "shape1/arktree-nested",
			rel:        "generated.decls.arktree.components.aicrew.bt.sub.combat_far_target..arktree.decl",
			wantBegin:  true, wantAssign: true},
		{name: "shape1/kiscule",
			rel:        "generated.decls.kiscule.models.kiscule_decls.flow_control.test_player..kiscule.decl",
			wantBegin:  true, wantAssign: true},
		{name: "shape1/midnightscene",
			rel:        "generated.decls.midnightscene.midnight.gameplay.player.base.combat.sword_slide_slash_right..midnightscene.decl",
			wantBegin:  true},

		// Shape 2 — animset. Whitespace-separated keys, quoted-string values,
		// nested typed blocks (alias { … }, event "name" { … }), scoped names
		// (`arkMidnightAnimMarker::Name_t`).
		{name: "shape2/timeshift",
			rel:        "generated.decls.animset.models.weapons.timeshift_device.timeshift..animset.decl",
			wantBegin:  true, wantTypBlk: true, wantAssign: true},
		{name: "shape2/sword_patrol-with-scoped-names",
			rel:        "generated.decls.animset.models.characters.female._animations_.low.female_low_sword_patrol..animset.decl",
			wantBegin:  true, wantTypBlk: true, wantAssign: true},

		// Shape 3 — material. Tab-aligned, bare-path values, namespaced keys,
		// brace-list tuples, function-call values.
		{name: "shape3/void_rock_01",
			rel:        "generated.decls.material.models.environment.voidhouse.rock_set.void_rock_01..material.decl",
			wantBegin:  true, wantTypBlk: true, wantAssign: true},

		// Shape 5 — md6def. Folded into Shape 2 walker; positional integer
		// names (`lod 0 { … }`).
		{name: "shape5/docker_small_01_head",
			rel:        "doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl",
			wantBegin:  true, wantTypBlk: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "golden", filepath.FromSlash(tc.rel))
			src, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("corpus file unavailable: %v", err)
			}
			toks, scanDiags, _ := scan.Scan(src)
			require.Empty(t, scanDiags, "%s: scan-clean precondition", tc.rel)

			rec := newRecorder(src)
			diags := parse.Walk(path, src, toks, rec)
			assert.Empty(t, diags, "%s: expected zero parse diagnostics, got %d:\n%v",
				tc.rel, len(diags), diags)

			if tc.wantBegin {
				assert.NotEmpty(t, filterKind(rec.events, "ObjectBegin"),
					"%s: expected at least one ObjectBegin event", tc.rel)
			}
			if tc.wantTypBlk {
				assert.NotEmpty(t, filterKind(rec.events, "TypedBlock"),
					"%s: expected at least one TypedBlock event", tc.rel)
			}
			if tc.wantAssign {
				assert.NotEmpty(t, filterKind(rec.events, "Assignment"),
					"%s: expected at least one Assignment event", tc.rel)
			}
		})
	}
}

// TestWalk_TupleValue_ParsesAsAssignment — Shape-1 tuple value
// (`color = ( 1, 1, 1, 1 );`) emits an Assignment with ValTuple kind and
// no PARSE diagnostics. Inline so the assertion isn't lost in corpus noise.
func TestWalk_TupleValue_ParsesAsAssignment(t *testing.T) {
	src := []byte("{\n\tcolor = ( 1, 1, 1, 1 );\n}\n")
	toks, scanDiags, _ := scan.Scan(src)
	require.Empty(t, scanDiags)

	rec := newRecorder(src)
	diags := parse.WalkEntities(src, toks, rec)
	require.Empty(t, diags)

	asg := filterKind(rec.events, "Assignment")
	require.Len(t, asg, 1)
	a := asg[0].data.(evAssignment)
	assert.Equal(t, "color", a.keyBase)
	assert.Equal(t, parse.ValTuple, a.valueKind)
	assert.True(t, a.hasSemi, "tuple values still require trailing ';'")
}
