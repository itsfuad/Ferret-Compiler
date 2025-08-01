package parser

import (
	"testing"
)

func TestIfStatementParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
		desc    string
	}{
		{
			name: "Simple if statement",
			input: `if x > 0 {
				return x;
			}`,
			isValid: true,
			desc:    "Basic if statement with a condition and block",
		},
		{
			name: "If statement with parentheses",
			input: `if (x > 0) {
				return x;
			}`,
			isValid: true,
			desc:    "If statement with parenthesized condition",
		},
		{
			name: "If-else statement",
			input: `if x > 0 {
				return x;
			} else {
				return -x;
			}`,
			isValid: true,
			desc:    "If statement with else branch",
		},
		{
			name: "If-else-if statement",
			input: `if x > 0 {
				return x;
			} else if x < 0 {
				return -x;
			} else {
				return 0;
			}`,
			isValid: true,
			desc:    "If statement with else-if and else branches",
		},
		{
			name: "Multiple else-if branches",
			input: `if x > 10 {
				return "high";
			} else if x > 5 {
				return "medium";
			} else if x > 0 {
				return "low";
			} else {
				return "zero or negative";
			}`,
			isValid: true,
			desc:    "If statement with multiple else-if branches",
		},
		{
			name:    "Missing condition",
			input:   "if { return x; }",
			isValid: false,
			desc:    "If statement without a condition should fail",
		},
		{
			name:    "Missing body",
			input:   "if x > 0",
			isValid: false,
			desc:    "If statement without a body should fail",
		},
		{
			name:    "Invalid else placement",
			input:   "if x > 0 { return x; } else if { return -x; }",
			isValid: false,
			desc:    "Else-if without condition should fail",
		},
		{
			name:    "Missing else-if condition",
			input:   "if x > 0 { return x; } else if { return -x; }",
			isValid: false,
			desc:    "Else-if without condition should fail",
		},
		{
			name:    "Missing else body",
			input:   "if x > 0 { return x; } else",
			isValid: false,
			desc:    "Else without body should fail",
		},
		{
			name: "Complex condition",
			input: `if x > 0 && y < 10 || z == 5 {
				return true;
			}`,
			isValid: true,
			desc:    "If statement with complex logical condition",
		},
		{
			name: "Nested if statements",
			input: `if x > 0 {
				if y > 0 {
					return x + y;
				} else {
					return x - y;
				}
			}`,
			isValid: true,
			desc:    "Nested if statements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
