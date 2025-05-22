package parser

import (
	"testing"
)

func TestFunctionParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
		desc    string
	}{
		{
			name: "Simple function declaration",
			input: `fn add(a: i32, b: i32) -> i32 {
				return a + b;
			}`,
			isValid: true,
			desc:    "Basic function with two parameters and single return type",
		},
		{
			name: "Function with multiple return types",
			input: `fn div(a: i32, b: i32) -> (i32, bool) {
				return a / b, true;
			}`,
			isValid: true,
			desc:    "Function with multiple return types in parentheses",
		},
		{
			name: "Anonymous function assignment",
			input: `const add = fn(a: i32, b: i32) -> i32 {
				return a + b;
			};`,
			isValid: true,
			desc:    "Anonymous function assigned to a constant",
		},
		{
			name: "Function without return type",
			input: `fn greet(name: str) {
				return;
			}`,
			isValid: true,
			desc:    "Function without explicit return type",
		},
		{
			name: "Function with empty parameter list",
			input: `fn init() -> bool {
				return true;
			}`,
			isValid: true,
			desc:    "Function with no parameters",
		},
		{
			name:    "Missing parameter type",
			input:   "fn add(a, b) -> i32 { return a + b; }",
			isValid: false,
			desc:    "Function with missing parameter types should fail",
		},
		{
			name:    "Missing parameter name",
			input:   "fn add(:i32, :i32) -> i32 { return 0; }",
			isValid: false,
			desc:    "Function with missing parameter names should fail",
		},
		{
			name:    "Invalid return type syntax",
			input:   "fn add(a: i32, b: i32) -> (i32,) { return 0; }",
			isValid: true,
			desc:    "Function with trailing comma in return type list should pass with a warning",
		},
		{
			name:    "Missing function body",
			input:   "fn add(a: i32, b: i32) -> i32;",
			isValid: false,
			desc:    "Function without body should fail",
		},
		{
			name: "Function with complex return types",
			input: `fn process(data: []i32) -> ([]i32, str, bool) {
				return data, "ok", true;
			}`,
			isValid: true,
			desc:    "Function with multiple complex return types",
		},
		{
			name: "Nested function declaration",
			input: `fn outer() -> fn(x: i32) -> i32 {
				return fn(x: i32) -> i32 {
					return x * 2;
				};
			}`,
			isValid: true,
			desc:    "Function with nested function declaration",
		},
		{
			name:    "Invalid parameter list",
			input:   "fn bad(a: i32,) -> i32 { return a; }",
			isValid: true,
			desc:    "Function with trailing comma in parameter list should pass with a warning",
		},
		{
			name:    "Trailing comma in function return type",
			input:   "fn add(a: i32, b: i32) -> (i32, f32,) { return a + b, 1.0; }",
			isValid: true,
			desc:    "Function with trailing comma in return type should pass with a warning",
		},
		{
			name:    "Function call with no arguments",
			input:   "hello();",
			isValid: true,
			desc:    "Function call with no arguments",
		},
		{
			name:    "Function call with 1 argument",
			input:   "hello(1);",
			isValid: true,
			desc:    "Function call with 1 argument",
		},
		{
			name:    "Function call with 2 arguments, missing semicolon",
			input:   "hello(1, 2)",
			isValid: false,
			desc:    "Function call with 2 arguments should fail",
		},
		{
			name:    "Function call with 2 arguments",
			input:   "hello(1, 2);",
			isValid: true,
			desc:    "Function call with 2 arguments",
		},
		{
			name:    "Function call with trailing comma",
			input:   "hello(1, 2, 4,);",
			isValid: true,
			desc:    "Function call with trailing comma should pass with a warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
