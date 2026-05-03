package parse_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"void-slice/internal/parse"
	"void-slice/internal/scan"
)

// -------------------------
// Test handler
// -------------------------

type event struct {
	kind string
	data any
}

type evVersion struct {
	value int64
}
type evComponentBegin struct{}
type evComponentDecl struct {
	typeLex string
	nameLex string
}
type evComponentEnd struct{}
type evObjectBegin struct{}
type evObjectEnd struct{}
type evAssignment struct {
	keyBase    string
	indexers   []parse.Indexer
	valueKind  parse.ValueKind
	valueLex   string // for scalar values
	hasSemi    bool
}
type evTypedBlock struct {
	typeLex string
	nameLex string // empty for bare ident { ... }
}
type evDiag struct {
	code scan.DiagnosticCode
}

type recorder struct {
	src    []byte
	events []event
	diags  []scan.Diagnostic
}

func newRecorder(src []byte) *recorder { return &recorder{src: src} }

func (r *recorder) lex(t scan.Token) string {
	if t.Span.Start == t.Span.End {
		return ""
	}
	return string(r.src[t.Span.Start:t.Span.End])
}

func (r *recorder) OnVersion(versionTok scan.Token, value int64) {
	r.events = append(r.events, event{"Version", evVersion{value}})
}
func (r *recorder) OnComponentBegin(componentTok, lbrace scan.Token) {
	r.events = append(r.events, event{"ComponentBegin", evComponentBegin{}})
}
func (r *recorder) OnComponentDecl(typeTok, nameTok, lbrace scan.Token) {
	r.events = append(r.events, event{"ComponentDecl", evComponentDecl{r.lex(typeTok), r.lex(nameTok)}})
}
func (r *recorder) OnComponentEnd(rbrace scan.Token) {
	r.events = append(r.events, event{"ComponentEnd", evComponentEnd{}})
}
func (r *recorder) OnObjectBegin(lbrace scan.Token) {
	r.events = append(r.events, event{"ObjectBegin", evObjectBegin{}})
}
func (r *recorder) OnObjectEnd(rbrace scan.Token) {
	r.events = append(r.events, event{"ObjectEnd", evObjectEnd{}})
}
func (r *recorder) OnAssignment(key parse.Key, eqTok scan.Token, value parse.Value, semiTok scan.Token) {
	var valueLex string
	if value.Kind != parse.ValObject {
		valueLex = r.lex(value.Tok)
	}
	r.events = append(r.events, event{"Assignment", evAssignment{
		keyBase:   r.lex(key.BaseTok),
		indexers:  key.Indexers,
		valueKind: value.Kind,
		valueLex:  valueLex,
		hasSemi:   semiTok.Span.Start != semiTok.Span.End,
	}})
}
func (r *recorder) OnTypedBlock(typeTok, nameTok, lbrace scan.Token) {
	r.events = append(r.events, event{"TypedBlock", evTypedBlock{r.lex(typeTok), r.lex(nameTok)}})
}
func (r *recorder) OnDiag(diag scan.Diagnostic) {
	r.diags = append(r.diags, diag)
	r.events = append(r.events, event{"Diag", evDiag{diag.Code}})
}

// scan + walk helpers

func scanAndWalk(t *testing.T, src []byte) (*recorder, []scan.Diagnostic) {
	t.Helper()
	toks, scanDiags, _ := scan.Scan(src)
	require.Empty(t, scanDiags, "unexpected scan diagnostics")
	rec := newRecorder(src)
	parseDiags := parse.WalkEntities(src, toks, rec, parse.Opts{})
	return rec, parseDiags
}

func eventKinds(rec *recorder) []string {
	out := make([]string, len(rec.events))
	for i, e := range rec.events {
		out[i] = e.kind
	}
	return out
}

// -------------------------
// Happy path tests
// -------------------------

func TestVersion(t *testing.T) {
	src := []byte("Version 42\n") // trailing newline avoids "unterminated number" scan diag
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)
	require.Len(t, rec.events, 1)
	assert.Equal(t, "Version", rec.events[0].kind)
	assert.Equal(t, int64(42), rec.events[0].data.(evVersion).value)
}

func TestComponentBeginEnd(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)
	assert.Equal(t, []string{
		"Version", "ComponentBegin", "ComponentDecl", "ComponentEnd",
	}, eventKinds(rec))

	decl := rec.events[2].data.(evComponentDecl)
	assert.Equal(t, "cpntFoo", decl.typeLex)
	assert.Equal(t, "myFoo", decl.nameLex)
}

func TestAssignmentScalar(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		m_speed = 1.5f;
		m_name = "hello";
		m_flag = true;
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)

	assignments := filterKind(rec.events, "Assignment")
	require.Len(t, assignments, 3)

	a0 := assignments[0].data.(evAssignment)
	assert.Equal(t, "m_speed", a0.keyBase)
	assert.Equal(t, parse.ValNumber, a0.valueKind)
	assert.Equal(t, "1.5f", a0.valueLex)
	assert.True(t, a0.hasSemi)

	a1 := assignments[1].data.(evAssignment)
	assert.Equal(t, "m_name", a1.keyBase)
	assert.Equal(t, parse.ValString, a1.valueKind)
	assert.Equal(t, `"hello"`, a1.valueLex)

	a2 := assignments[2].data.(evAssignment)
	assert.Equal(t, "m_flag", a2.keyBase)
	assert.Equal(t, parse.ValIdent, a2.valueKind)
	assert.Equal(t, "true", a2.valueLex)
}

func TestAssignmentObjectValue(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		edit = {
			m_x = 1;
		}
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)

	assert.Equal(t, []string{
		"Version",
		"ComponentBegin",
		"ComponentDecl",
		"ObjectBegin",  // edit value
		"Assignment",   // m_x
		"ObjectEnd",
		"Assignment",   // edit =
		"ComponentEnd",
	}, eventKinds(rec))

	// Inner assignments (inside the object value) fire before the outer assignment.
	// So assignments[0] = m_x, assignments[1] = edit.
	assignments := filterKind(rec.events, "Assignment")
	require.Len(t, assignments, 2)
	editAssign := assignments[1]
	a := editAssign.data.(evAssignment)
	assert.Equal(t, "edit", a.keyBase)
	assert.Equal(t, parse.ValObject, a.valueKind)
	assert.False(t, a.hasSemi, "object values have no semicolon")
}

func TestIndexerInt(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		item[0] = {
			m_val = 1;
		}
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)

	assignments := filterKind(rec.events, "Assignment")
	// Inner assignments fire before outer: assignments[0]=m_val, assignments[1]=item[0]
	require.Len(t, assignments, 2)
	itemAssign := assignments[1].data.(evAssignment)
	assert.Equal(t, "item", itemAssign.keyBase)
	require.Len(t, itemAssign.indexers, 1)
	assert.Equal(t, parse.IndexInt, itemAssign.indexers[0].Kind)
	assert.Equal(t, int64(0), itemAssign.indexers[0].IntValue)
}

func TestIndexerString(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		item_add["abc"] = 99;
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)

	assignments := filterKind(rec.events, "Assignment")
	require.Len(t, assignments, 1)
	a := assignments[0].data.(evAssignment)
	assert.Equal(t, "item_add", a.keyBase)
	require.Len(t, a.indexers, 1)
	assert.Equal(t, parse.IndexString, a.indexers[0].Kind)
}

func TestTypedBlockTwoIdent(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		innerType innerName {
			m_x = 1;
		}
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)

	blocks := filterKind(rec.events, "TypedBlock")
	require.Len(t, blocks, 1)
	b := blocks[0].data.(evTypedBlock)
	assert.Equal(t, "innerType", b.typeLex)
	assert.Equal(t, "innerName", b.nameLex)
}

func TestTypedBlockBareIdent(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		edit {
			m_x = 1;
		}
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)

	blocks := filterKind(rec.events, "TypedBlock")
	require.Len(t, blocks, 1)
	b := blocks[0].data.(evTypedBlock)
	assert.Equal(t, "edit", b.typeLex)
	assert.Equal(t, "", b.nameLex, "bare block has no name token")
}

func TestNestedObjects(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		edit = {
			m_rules = {
				num = 1;
			}
		}
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)

	begins := filterKind(rec.events, "ObjectBegin")
	ends := filterKind(rec.events, "ObjectEnd")
	assert.Len(t, begins, 2, "two nested object begins")
	assert.Len(t, ends, 2, "two nested object ends")
}

func TestMultipleComponents(t *testing.T) {
	src := []byte(`Version 6
component {
	cpntA nameA {
		m_x = 1;
	}
}
component {
	cpntB nameB {
		m_y = 2;
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Empty(t, diags)

	decls := filterKind(rec.events, "ComponentDecl")
	require.Len(t, decls, 2)
	assert.Equal(t, "cpntA", decls[0].data.(evComponentDecl).typeLex)
	assert.Equal(t, "cpntB", decls[1].data.(evComponentDecl).typeLex)
}

// -------------------------
// Error path tests
// -------------------------

func TestMissingSemicolon(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		m_x = 1
		m_y = 2;
	}
}`)
	rec, diags := scanAndWalk(t, src)
	require.Len(t, diags, 1)
	assert.Equal(t, parse.Codes.EXPECTED_SEMICOLON, diags[0].Code)

	// parsing must continue after the error; m_y should still be captured
	assignments := filterKind(rec.events, "Assignment")
	keys := make([]string, len(assignments))
	for i, a := range assignments {
		keys[i] = a.data.(evAssignment).keyBase
	}
	assert.Contains(t, keys, "m_y", "parser should continue after missing semicolon")
}

func TestUnterminatedObject(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		edit = {
			m_x = 1;
`)
	toks, _, _ := scan.Scan(src)
	rec := newRecorder(src)
	diags := parse.WalkEntities(src, toks, rec, parse.Opts{})
	assert.NotEmpty(t, diags, "expected diagnostics for unterminated object")
}

func TestUnexpectedTopLevelToken(t *testing.T) {
	src := []byte(`Version 1
garbage
component {
	cpntFoo myFoo {
	}
}`)
	rec, diags := scanAndWalk(t, src)
	assert.NotEmpty(t, diags)
	diagKinds := filterKind(rec.events, "Diag")
	assert.NotEmpty(t, diagKinds)

	// component after the garbage token should still parse
	decls := filterKind(rec.events, "ComponentDecl")
	assert.NotEmpty(t, decls, "parser should recover and parse valid component")
}

// -------------------------
// Integration test
// -------------------------

func TestIntegration_EntitiesGoldenFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "corpus-mini", "d2", "game1",
		"maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("golden file not available: %v", err)
	}

	toks, scanDiags, _ := scan.Scan(src)
	require.Empty(t, scanDiags, "golden file must be scan-clean")

	rec := newRecorder(src)
	parseDiags := parse.WalkEntities(src, toks, rec, parse.Opts{})
	assert.Empty(t, parseDiags,
		"golden entities file must produce zero parse diagnostics; got %d:\n%v",
		len(parseDiags), parseDiags)

	// basic sanity: we should see Version + at least one component
	assert.NotEmpty(t, filterKind(rec.events, "Version"))
	assert.NotEmpty(t, filterKind(rec.events, "ComponentBegin"))
}

// -------------------------
// Diagnostic-count cap (M7.1)
// -------------------------

// garbageSrc returns a stream of n top-level identifiers separated by spaces.
// At top level, each unknown identifier not followed by '{' produces exactly
// one PARSE_UNEXPECTED_TOKEN diagnostic — so n garbage idents → n diagnostics.
func garbageSrc(n int) []byte {
	return []byte(strings.Repeat("garbage ", n))
}

func TestDiagnosticCap_OptIn(t *testing.T) {
	const cap = 5
	src := garbageSrc(20)
	toks, _, _ := scan.Scan(src)
	rec := newRecorder(src)
	diags := parse.WalkEntities(src, toks, rec, parse.Opts{MaxDiagnostics: cap})

	require.Len(t, diags, cap, "diags slice must equal the cap")
	assert.Equal(t, parse.Codes.DIAGNOSTICS_TRUNCATED, diags[cap-1].Code,
		"final entry must be the truncation sentinel")
	for i, d := range diags[:cap-1] {
		assert.NotEqual(t, parse.Codes.DIAGNOSTICS_TRUNCATED, d.Code,
			"non-final entries must be original errors (got sentinel at i=%d)", i)
	}

	// Handler should also have observed exactly one sentinel forwarded via OnDiag.
	sentinels := 0
	for _, d := range rec.diags {
		if d.Code == parse.Codes.DIAGNOSTICS_TRUNCATED {
			sentinels++
		}
	}
	assert.Equal(t, 1, sentinels, "handler should see the sentinel exactly once")
}

func TestDiagnosticCap_DefaultUncapped(t *testing.T) {
	const n = 20
	src := garbageSrc(n)
	toks, _, _ := scan.Scan(src)
	rec := newRecorder(src)
	diags := parse.WalkEntities(src, toks, rec, parse.Opts{})

	assert.Equal(t, n, len(diags), "default Opts must not cap diagnostics")
	for _, d := range diags {
		assert.NotEqual(t, parse.Codes.DIAGNOSTICS_TRUNCATED, d.Code,
			"no truncation sentinel when uncapped")
	}
}

// -------------------------
// Helpers
// -------------------------

func filterKind(events []event, kind string) []event {
	var out []event
	for _, e := range events {
		if e.kind == kind {
			out = append(out, e)
		}
	}
	return out
}
