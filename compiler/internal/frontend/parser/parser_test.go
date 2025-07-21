package parser

import (
	"testing"
)

func TestParserBasics(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		{"let x = 42;", true, "Variable declaration with initialization"},
		{"const y = true;", true, "Constant declaration"},
		{"x = 42; y = 10;", true, "Multiple statements"},
		{"let x;", false, "Declaration without initialization"},
		{"let x: i32; x = 42;", true, "Declaration followed by assignment"},
		{"let x, y: i32;\nx, y = 1, 2;", true, "Multiple declaration and assignment"},
		{"let x: i32 = 42;", true, "Typed variable declaration"},
		{"", false, "Empty file"},
		{"a;", true, "Single unused statement"},
		{"a, b;", true, "Multiple unused statements"},
		{"let", false, "Incomplete declaration"},
		{"x =", false, "Incomplete assignment"},
		{"let x = ;", false, "Missing expression in declaration"},
		{"let x = 42; x = ;", false, "Missing expression in assignment"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
