package parser

import (
	"testing"
)

func TestMethodParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
		desc    string
	}{
		{
			name: "Simple method declaration",
			input: `fn (r: Receiver) add(a: i32, b: i32) -> i32 {
				return a + b;
			}`,
			isValid: true,
			desc:    "Basic method with two parameters and single return type",
		},
		{
			name: "Method with multiple parameters",
			input: `fn (r: Receiver) add(a: i32, b: i32, c: i32) -> i32 {
				return a + b + c;
			}`,
			isValid: true,
			desc:    "Method with multiple parameters",
		},
		{
			name: "Method with multiple returns",
			input: `fn (r: Receiver) someMethod() -> (i32, i32) {
				return 1, 2;
			}`,
			isValid: true,
			desc:    "Method with multiple returns",
		},

		// invalid
		{
			name: "Method with no receiver",
			input: `fn () someMethod() -> i32 {
				return 1;
			}`,
			isValid: false,
			desc:    "Method with no receiver",
		},
		{
			name: "Method with multiple receivers",
			input: `fn (r: Receiver, r2: Receiver) someMethod() -> i32 {
				return 1;
			}`,
			isValid: true,
			desc:    "Method with multiple receivers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
