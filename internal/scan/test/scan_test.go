package scan_test

import (
	"testing"
	"void-slice/internal/scan"

	"github.com/stretchr/testify/assert"
)

func TestIsASCIIPunct(t *testing.T) {
	type tc struct {
		name string
		b    byte
		want bool
	}

	tests := []tc{
		// --- Lower edge cases / non-printing ---
		{name: "NUL", b: 0x00, want: false},
		{name: "space", b: ' ', want: false}, // 0x20
		{name: "DEL", b: 0x7F, want: false},  // 127, not in any range below
		{name: "non_ascii_128", b: 0x80, want: false},

		// --- Digits / letters should be false ---
		{name: "digit_0", b: '0', want: false},
		{name: "digit_9", b: '9', want: false},
		{name: "upper_A", b: 'A', want: false},
		{name: "upper_Z", b: 'Z', want: false},
		{name: "lower_a", b: 'a', want: false},
		{name: "lower_z", b: 'z', want: false},

		// --- Range 1: '!'..'/' (33..47) ---
		{name: "bang_lower_bound", b: '!', want: true},
		{name: "paren_open", b: '(', want: true},
		{name: "paren_close", b: ')', want: true},
		{name: "plus", b: '+', want: true},
		{name: "slash_upper_bound", b: '/', want: true},

		// --- Range 2: ':'..'@' (58..64) ---
		{name: "colon_lower_bound", b: ':', want: true},
		{name: "semicolon", b: ';', want: true},
		{name: "equals", b: '=', want: true},
		{name: "at_upper_bound", b: '@', want: true},

		// --- Range 3: '['..'`' (91..96) ---
		{name: "lbracket_lower_bound", b: '[', want: true},
		{name: "backslash", b: '\\', want: true},
		{name: "rbracket", b: ']', want: true},
		{name: "caret", b: '^', want: true},
		{name: "underscore", b: '_', want: true},
		{name: "backtick_upper_bound", b: '`', want: true},

		// --- Range 4: '{'..'~' (123..126) ---
		{name: "lbrace_lower_bound", b: '{', want: true},
		{name: "pipe", b: '|', want: true},
		{name: "rbrace", b: '}', want: true},
		{name: "tilde_upper_bound", b: '~', want: true},

		// --- Just outside boundaries (to prove bounds are tight) ---
		{name: "one_before_bang", b: 0x20, want: false}, // space
		{name: "one_after_tilde", b: 0x7F, want: false}, // DEL
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scan.IsASCIIPunct(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
