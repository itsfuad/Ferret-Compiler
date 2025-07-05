package parser

import (
	"testing"
)

func TestParseAssignment(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		{"x = 42;", true, "Single variable assignment"},
		{"x, y = 42, 3.14;", true, "Multiple variable assignment"},
		{"x, y, z = 1, 2, 3;", true, "Three variable assignment"},
		{"x = \"hello\";", true, "String assignment"},
		{"x, y = 10, true;", true, "Mixed type assignment"},
		{"x, y = someVal;", true, "Shared value assigned to multiple variables"},
		{"x = 1, 2;", false, "More values than variables"},
		{"x, y = ;", false, "Missing values in assignment"},
		{"x, y = 1", false, "Missing semicolon"},
		{"x, = 1, 2;", false, "Invalid variable list"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
