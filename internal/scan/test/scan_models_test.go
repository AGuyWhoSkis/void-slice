package scan_test

import (
	"testing"
	"void-slice/internal/scan"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLineIndex(t *testing.T) {
	// BuildLineIndex returns LineIndex.Newlines as the 0-based byte offsets of '\n' in src.

	tcs := []struct {
		name           string
		inputStr       string
		expectNewlines []int
	}{
		{
			name:           "only newlines",
			inputStr:       "\n\n\n\n\n",
			expectNewlines: []int{0, 1, 2, 3, 4},
		},
		{
			name:           "no newlines",
			inputStr:       "00000",
			expectNewlines: []int{},
		},
		{
			name:           "mixed content",
			inputStr:       "a\nbc\n\nxyz",
			expectNewlines: []int{1, 4, 5},
		},
		{
			name:           "ends with newline",
			inputStr:       "abc\n",
			expectNewlines: []int{3},
		},
		{
			name:           "empty input",
			inputStr:       "",
			expectNewlines: []int{},
		},
	}

	for _, tc := range tcs {
		tc := tc // capture
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := scan.BuildLineIndex([]byte(tc.inputStr))

			require.Equal(t, tc.expectNewlines, got.Newlines)
		})
	}
}

func TestSpanPos(t *testing.T) {
	li := scan.LineIndex{Newlines: []int{10, 20, 30, 40, 50}}

	out := li.SpanPos(scan.Span{Start: 1, End: 20})

	assert.Equal(t, scan.Pos{Line: 1, Col: 2}, out.Start)
	assert.Equal(t, scan.Pos{Line: 2, Col: 10}, out.End)
}

// func (li LineIndex) PosAt(offset int) Pos {

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func TestPosAt(t *testing.T) {
	tcs := []struct {
		name      string
		input     string
		expectNLs []int
		cases     []struct {
			offset int
			line   int
			col    int
		}
	}{
		{
			name:      "no newlines",
			input:     "ABCDEFGHIJKL$NOPQRSTUVWXYZ",
			expectNLs: []int{},
			cases: []struct {
				offset int
				line   int
				col    int
			}{
				{offset: 0, line: 1, col: 1},
				{offset: 12, line: 1, col: 13},
				{offset: len("ABCDEFGHIJKL$NOPQRSTUVWXYZ"), line: 1, col: len("ABCDEFGHIJKL$NOPQRSTUVWXYZ") + 1}, // EOF pos
			},
		},
		{
			name:      "multiple newlines incl blank line",
			input:     "ABCDEFGH\nIJKLMN\n\nOPQRS$UVW\nXYZ",
			expectNLs: []int{8, 15, 16, 26}, // NOTE: verify these indices carefully
			cases: []struct {
				offset int
				line   int
				col    int
			}{
				{offset: 0, line: 1, col: 1},   // 'A'
				{offset: 8, line: 1, col: 9},   // '\n' itself
				{offset: 9, line: 2, col: 1},   // 'I'
				{offset: 15, line: 2, col: 7},  // '\n' after N
				{offset: 16, line: 3, col: 1},  // '\n' blank line
				{offset: 17, line: 4, col: 1},  // 'O'
				{offset: 22, line: 4, col: 6},  // '$' (your original check)
				{offset: 26, line: 4, col: 10}, // '\n' after W
				{offset: 27, line: 5, col: 1},  // 'X'
			},
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.input)
			li := scan.BuildLineIndex(src)

			require.Equal(t, tc.expectNLs, li.Newlines)

			for _, c := range tc.cases {
				got := li.PosAt(c.offset)
				require.Equal(t, c.line, got.Line, "offset=%d", c.offset)
				require.Equal(t, c.col, got.Col, "offset=%d", c.offset)
			}
		})
	}
}
