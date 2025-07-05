package parser

import (
	"testing"
)

func TestParseVarDecl(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		{"let x;", false, "Variable with no type"},
		{"let x, y;", false, "Variable with no type"},
		{"let x: i32;", true, "Variable with type annotation"},
		{"let x, y: i32, str;", true, "Multiple variables with type annotations"},
		{"let x = 42;", true, "Variable initialized with a value"},
		{"let x, y = 42, 3.14;", true, "Multiple variables initialized with values"},
		{"let x, y: i32 = 10, 20;", true, "Typed variables initialized"},
		{"let p, q: i32, str = 10, \"hello\";", true, "Multiple typed variables initialized"},
		{"let x, y = p;", true, "Mismatched variable and value count"},
		{"let x, y: i32;", true, "Shared type annotation"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
