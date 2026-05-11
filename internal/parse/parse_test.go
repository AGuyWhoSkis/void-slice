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
	parseDiags := parse.WalkEntities(src, toks, rec)
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
	diags := parse.WalkEntities(src, toks, rec)
	assert.NotEmpty(t, diags, "expected diagnostics for unterminated object")
}

// TestM12_7_QuoteHidesBrace pins that a `}` inside a QUOTE_LITERAL never
// reaches the parser — the indent-aware close helper added in M12.7 must
// not be confused by braces hidden inside string values. Regression contract.
func TestM12_7_QuoteHidesBrace(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		edit = {
			m_x = "}";
		}
	}
}
`)
	_, diags := scanAndWalk(t, src)
	assert.Empty(t, diags, "quote-hidden `}` must not trip indent-aware close")
}

// TestM12_7_BlockCommentHidesBrace pins that `}` inside COMMENT_BLOCK and
// COMMENT_LINE tokens never reaches the parser. Same regression contract as
// the quote case.
func TestM12_7_BlockCommentHidesBrace(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		edit = {
			m_x = 1; /* } */
			m_y = 2; // }
		}
	}
}
`)
	_, diags := scanAndWalk(t, src)
	assert.Empty(t, diags, "comment-hidden `}` must not trip indent-aware close")
}

// TestM12_7_MidFileMissingBraceAnchorsAtInner is the parser-layer mirror of
// the locality fixture: the ticket's 13-line sample as in-source bytes. The
// parser must emit exactly one PARSE_UNTERMINATED_OBJECT anchored at the
// inner `{` on line 9 — the EOF-anchored cascade from greedy brace matching
// is what M12.7 fixes.
func TestM12_7_MidFileMissingBraceAnchorsAtInner(t *testing.T) {
	src := []byte("Version 1\n" +
		"component {\n" +
		"\tcpntTest myTest {\n" +
		"\t\tedit = {\n" +
		"\t\t\tm_items = {\n" +
		"\t\t\t\tnum = 2;\n" +
		"\t\t\t\titem[0] = { m_val = \"a\"; }\n" +
		"\t\t\t\titem[1] = { m_val = \"b\"; }\n" +
		"\t\t\t\titem[2] = { m_val = \"c\";\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t}\n" +
		"}\n")

	_, diags := scanAndWalk(t, src)

	var unterm []scan.Diagnostic
	for _, d := range diags {
		if d.Code == parse.Codes.UNTERMINATED_OBJECT {
			unterm = append(unterm, d)
		}
	}
	require.Len(t, unterm, 1, "expected exactly one PARSE_UNTERMINATED_OBJECT, got diags: %v", diags)

	li := scan.BuildLineIndex(src)
	pos := li.PosAt(unterm[0].Span.Start)
	assert.Equal(t, 9, pos.Line, "PARSE_UNTERMINATED_OBJECT must anchor at line 9; got %d:%d (%v)", pos.Line, pos.Col, diags)
}

// TestM12_16_IndentInvariantCascade pins the M12.16 contract: the
// whitespace-cascade fixture (a balanced file whose only fault is a
// mid-line unterminated quote) must produce the same diagnostic-code
// multiset whether each line carries its committed leading tabs (V1) or
// every line is flush-left (V2). Before M12.16 the parser's
// indent-gated close emitted 7 cascade diagnostics for V1 and 1 for V2 —
// pure layout-dependence on identical logical input. After the fix the
// balance gate refuses to re-anchor when the file is structurally
// balanced, so both forms reduce to the single VOID_SCAN at the broken
// quote.
func TestM12_16_IndentInvariantCascade(t *testing.T) {
	v1 := []byte("Version 1\n" +
		"component {\n" +
		"\tcpntTest myTest {\n" +
		"\t\tedit = {\n" +
		"\t\t\tm_items = {\n" +
		"\t\t\t\tnum = 2;\n" +
		"\t\t\t\titem[0] = { m_val = \"a\"; }\n" +
		"\t\t\t\titem[1] = { x_val = \"b;\n" +
		"}\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t}\n" +
		"}\n")
	v2 := stripLeadingWhitespace(v1)

	codes1 := walkAndCount(t, v1)
	codes2 := walkAndCount(t, v2)
	assert.Equal(t, codes1, codes2,
		"V1 and V2 must produce identical diagnostic-code multisets; got V1=%v V2=%v", codes1, codes2)
	assert.Equal(t, map[scan.DiagnosticCode]int{scan.Codes.SCAN: 1}, codes1,
		"V1 must reduce to a single VOID_SCAN; got %v", codes1)
}

// stripLeadingWhitespace produces the lexinvariance V2 form: drop tabs
// and spaces at the start of every line, leaving inter-token bytes
// unchanged so the lexical token stream is identical to V1.
func stripLeadingWhitespace(src []byte) []byte {
	var out []byte
	atLineStart := true
	for _, b := range src {
		if atLineStart && (b == ' ' || b == '\t') {
			continue
		}
		out = append(out, b)
		atLineStart = b == '\n'
	}
	return out
}

func walkAndCount(t *testing.T, src []byte) map[scan.DiagnosticCode]int {
	t.Helper()
	toks, scanDiags, _ := scan.Scan(src)
	rec := newRecorder(src)
	parseDiags := parse.WalkEntities(src, toks, rec)
	counts := map[scan.DiagnosticCode]int{}
	for _, d := range scanDiags {
		counts[d.Code]++
	}
	for _, d := range parseDiags {
		counts[d.Code]++
	}
	return counts
}

// TestM12_8_MissingOpenBraceAfterEqAnchorsAtAssignment pins the parser-layer
// contract for the M12.8 cascade fixture: `edit =` with the opening `{`
// forgotten must produce exactly one focused PARSE_EXPECTED_SYMBOL anchored at
// line 4's `=`, not the five-diag scatter through lines 5/12/13 that today's
// fall-through recovery produces.
func TestM12_8_MissingOpenBraceAfterEqAnchorsAtAssignment(t *testing.T) {
	src := []byte("Version 1\n" +
		"component {\n" +
		"\tcpntTest myTest {\n" +
		"\t\tedit =\n" +
		"\t\t\tm_items = {\n" +
		"\t\t\t\tnum = 3;\n" +
		"\t\t\t\titem[0] = { m_val = \"a\"; }\n" +
		"\t\t\t\titem[1] = { m_val = \"b\"; }\n" +
		"\t\t\t\titem[2] = { m_val = \"c\"; }\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t}\n" +
		"}\n")

	_, diags := scanAndWalk(t, src)

	require.Len(t, diags, 1, "expected exactly one diagnostic; got: %v", diags)
	assert.Equal(t, parse.Codes.EXPECTED_SYMBOL, diags[0].Code, "diag code")

	li := scan.BuildLineIndex(src)
	pos := li.PosAt(diags[0].Span.Start)
	assert.Equal(t, 4, pos.Line, "diagnostic must anchor on line 4 (the `=`); got %d:%d (%v)", pos.Line, pos.Col, diags)
}

// TestM12_8_LegitimateIdentValueDoesNotTrigger is the negative-test mate to
// the M12.8 anchor test: a legitimate scalar-IDENT assignment followed by
// another statement (no missing brace anywhere) must not trip the new
// missing-`{`-after-`=` heuristic.
func TestM12_8_LegitimateIdentValueDoesNotTrigger(t *testing.T) {
	src := []byte(`Version 1
component {
	cpntFoo myFoo {
		m_flag = true;
		m_other = 1;
	}
}
`)
	_, diags := scanAndWalk(t, src)
	assert.Empty(t, diags, "legitimate ident-value statement must not trigger M12.8 heuristic")
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
// M12.12: per-call-site EOF anchor routing
// -------------------------
//
// Each test below uses a trailing newline so EOF-anchor and last-consumed-
// anchor land on different (line, col) coordinates — exposing the routing
// change. Each test asserts the diagnostic specific to that call site is
// anchored at the end of the last consumed token, not on the trailing-
// empty line that eofSpan would produce.

func findDiagByMessage(diags []scan.Diagnostic, want string) (scan.Diagnostic, bool) {
	for _, d := range diags {
		if d.Message == want {
			return d, true
		}
	}
	return scan.Diagnostic{}, false
}

// Site 1: expectKind for "expected version number after 'Version'".
func TestM12_12_ExpectKindAnchorsAtLastConsumed(t *testing.T) {
	src := []byte("Version\n")
	_, diags := scanAndWalk(t, src)
	d, ok := findDiagByMessage(diags, "expected version number after 'Version'")
	require.True(t, ok, "missing diagnostic; got: %v", diags)
	li := scan.BuildLineIndex(src)
	pos := li.PosAt(d.Span.Start)
	assert.Equal(t, 1, pos.Line, "anchor line")
	assert.Equal(t, 8, pos.Col, "anchor col (end of 'Version'); diags=%v", diags)
}

// Site 2: expectSym(']') for "expected ']' to close indexer".
func TestM12_12_ExpectSymCloseIndexerAnchorsAtLastConsumed(t *testing.T) {
	src := []byte("foo {\n\tx [ 0\n")
	_, diags := scanAndWalk(t, src)
	d, ok := findDiagByMessage(diags, "expected ']' to close indexer")
	require.True(t, ok, "missing diagnostic; got: %v", diags)
	li := scan.BuildLineIndex(src)
	pos := li.PosAt(d.Span.Start)
	assert.Equal(t, 2, pos.Line, "anchor line")
	assert.Equal(t, 7, pos.Col, "anchor col (end of '0'); diags=%v", diags)
}

// Site 3: direct emit for "expected index value inside '['".
func TestM12_12_ExpectedIndexValueAnchorsAtLastConsumed(t *testing.T) {
	src := []byte("foo {\n\tx [\n")
	_, diags := scanAndWalk(t, src)
	d, ok := findDiagByMessage(diags, "expected index value inside '['")
	require.True(t, ok, "missing diagnostic; got: %v", diags)
	li := scan.BuildLineIndex(src)
	pos := li.PosAt(d.Span.Start)
	assert.Equal(t, 2, pos.Line, "anchor line")
	assert.Equal(t, 5, pos.Col, "anchor col (end of '['); diags=%v", diags)
}

// Site 4: direct emit for "unexpected end of file in statement".
func TestM12_12_UnexpectedEOFInStatementAnchorsAtLastConsumed(t *testing.T) {
	src := []byte("foo {\n\tx\n")
	_, diags := scanAndWalk(t, src)
	d, ok := findDiagByMessage(diags, "unexpected end of file in statement")
	require.True(t, ok, "missing diagnostic; got: %v", diags)
	li := scan.BuildLineIndex(src)
	pos := li.PosAt(d.Span.Start)
	assert.Equal(t, 2, pos.Line, "anchor line")
	assert.Equal(t, 3, pos.Col, "anchor col (end of 'x'); diags=%v", diags)
}

// Site 5: direct emit for "expected value".
func TestM12_12_ExpectedValueAnchorsAtLastConsumed(t *testing.T) {
	src := []byte("foo {\n\tx =\n")
	_, diags := scanAndWalk(t, src)
	d, ok := findDiagByMessage(diags, "expected value")
	require.True(t, ok, "missing diagnostic; got: %v", diags)
	li := scan.BuildLineIndex(src)
	pos := li.PosAt(d.Span.Start)
	assert.Equal(t, 2, pos.Line, "anchor line")
	assert.Equal(t, 5, pos.Col, "anchor col (end of '='); diags=%v", diags)
}

// Site 6: direct emit for "unterminated tuple value".
func TestM12_12_UnterminatedTupleAnchorsAtLastConsumed(t *testing.T) {
	src := []byte("foo {\n\tx = ( a, b\n")
	_, diags := scanAndWalk(t, src)
	d, ok := findDiagByMessage(diags, "unterminated tuple value")
	require.True(t, ok, "missing diagnostic; got: %v", diags)
	li := scan.BuildLineIndex(src)
	pos := li.PosAt(d.Span.Start)
	assert.Equal(t, 2, pos.Line, "anchor line")
	assert.Equal(t, 12, pos.Col, "anchor col (end of 'b'); diags=%v", diags)
}

// -------------------------
// Integration test
// -------------------------

func TestIntegration_EntitiesGoldenFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "golden", "d2", "game1",
		"maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("golden file not available: %v", err)
	}

	toks, scanDiags, _ := scan.Scan(src)
	require.Empty(t, scanDiags, "golden file must be scan-clean")

	rec := newRecorder(src)
	parseDiags := parse.WalkEntities(src, toks, rec)
	assert.Empty(t, parseDiags,
		"golden entities file must produce zero parse diagnostics; got %d:\n%v",
		len(parseDiags), parseDiags)

	// basic sanity: we should see Version + at least one component
	assert.NotEmpty(t, filterKind(rec.events, "Version"))
	assert.NotEmpty(t, filterKind(rec.events, "ComponentBegin"))
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
