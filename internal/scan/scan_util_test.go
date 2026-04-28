package scan_test

import (
	"testing"
	"void-slice/internal/scan"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsWhitespace(t *testing.T) {
	type tc struct {
		b    byte
		want bool
	}
	tests := []tc{
		{' ', true},
		{'\t', true},
		{'\r', true},
		{'\v', true},
		{'\f', true},
		{'\n', false},
		{'a', false},
		{'0', false},
		{0x00, false},
		{0x1F, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, scan.IsWhitespace(tt.b), "byte %02x", tt.b)
	}
}

func TestIsDigit(t *testing.T) {
	type tc struct {
		b    byte
		want bool
	}
	tests := []tc{
		{'0', true},
		{'9', true},
		{'5', true},
		{'/', false}, // '0' - 1
		{':', false}, // '9' + 1
		{'a', false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, scan.IsDigit(tt.b), "byte %q", tt.b)
	}
}

func TestIsNumberTypeSuffix(t *testing.T) {
	type tc struct {
		b    byte
		want bool
	}
	tests := []tc{
		{'f', true},
		{'m', true},
		{'g', false},
		{'0', false},
		{'F', false},
		{'M', false},
		{' ', false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, scan.IsNumberTypeSuffix(tt.b), "byte %q", tt.b)
	}
}

func TestIsAlpha(t *testing.T) {
	type tc struct {
		b    byte
		want bool
	}
	tests := []tc{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'m', true},
		{'`', false}, // 'A' - 1
		{'{', false}, // 'Z' + 1
		{'`', false}, // 'a' - 1
		{'{', false}, // 'z' + 1
		{'0', false},
		{'_', false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, scan.IsAlpha(tt.b), "byte %q", tt.b)
	}
}

func TestIsIdentStart(t *testing.T) {
	type tc struct {
		b    byte
		want bool
	}
	tests := []tc{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'_', true},
		{'0', false},
		{'-', false},
		{' ', false},
		{'.', false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, scan.IsIdentStart(tt.b), "byte %q", tt.b)
	}
}

func TestIsIdentCont(t *testing.T) {
	type tc struct {
		b    byte
		want bool
	}
	tests := []tc{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'_', true},
		{'0', true},
		{'9', true},
		{'-', false},
		{'.', false},
		{' ', false},
		{'[', false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, scan.IsIdentCont(tt.b), "byte %q", tt.b)
	}
}

func TestLexeme(t *testing.T) {
	src := []byte("edit = 42")
	tok := scan.Token{Kind: scan.KindIdentifier, Span: scan.NewSpan(0, 4)}
	assert.Equal(t, []byte("edit"), scan.Lexeme(src, tok))

	sym := scan.Token{Kind: scan.KindSymbol, Span: scan.NewSpan(5, 6)}
	assert.Equal(t, []byte("="), scan.Lexeme(src, sym))

	num := scan.Token{Kind: scan.KindNumberLiteral, Span: scan.NewSpan(7, 9)}
	assert.Equal(t, []byte("42"), scan.Lexeme(src, num))
}

func TestSym(t *testing.T) {
	src := []byte("{ edit = 42 }")
	sym := scan.Token{Kind: scan.KindSymbol, Span: scan.NewSpan(0, 1)}
	assert.Equal(t, byte('{'), scan.Sym(src, sym))

	eq := scan.Token{Kind: scan.KindSymbol, Span: scan.NewSpan(7, 8)}
	assert.Equal(t, byte('='), scan.Sym(src, eq))
}

func TestEqIdent(t *testing.T) {
	src := []byte("edit = version")
	editTok := scan.Token{Kind: scan.KindIdentifier, Span: scan.NewSpan(0, 4)}
	verTok := scan.Token{Kind: scan.KindIdentifier, Span: scan.NewSpan(7, 14)}
	symTok := scan.Token{Kind: scan.KindSymbol, Span: scan.NewSpan(5, 6)}

	assert.True(t, scan.EqIdent(src, editTok, "edit"), "matching ident")
	assert.False(t, scan.EqIdent(src, editTok, "edix"), "same length, wrong chars")
	assert.False(t, scan.EqIdent(src, editTok, "ed"), "shorter lit")
	assert.False(t, scan.EqIdent(src, editTok, "editor"), "longer lit")
	assert.False(t, scan.EqIdent(src, symTok, "="), "wrong Kind returns false")
	assert.True(t, scan.EqIdent(src, verTok, "version"), "second ident")
}

func TestParseIntLiteral(t *testing.T) {
	src := []byte("42 -7 16f 1.5")
	tok42 := scan.Token{Kind: scan.KindNumberLiteral, Span: scan.NewSpan(0, 2)}
	tokNeg := scan.Token{Kind: scan.KindNumberLiteral, Span: scan.NewSpan(3, 5)}
	tokSuffix := scan.Token{Kind: scan.KindNumberLiteral, Span: scan.NewSpan(6, 9)}
	tokFloat := scan.Token{Kind: scan.KindNumberLiteral, Span: scan.NewSpan(10, 13)}

	v, err := scan.ParseIntLiteral(src, tok42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), v)

	v, err = scan.ParseIntLiteral(src, tokNeg)
	require.NoError(t, err)
	assert.Equal(t, int64(-7), v)

	v, err = scan.ParseIntLiteral(src, tokSuffix)
	require.NoError(t, err, "suffix 'f' should be stripped before parsing")
	assert.Equal(t, int64(16), v)

	_, err = scan.ParseIntLiteral(src, tokFloat)
	assert.Error(t, err, "float literal should fail to parse as int")
}
