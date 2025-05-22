package parser

import (
	"testing"
)

func TestExpressionParsing(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		{"let x = a + b;", true, "Binary addition"},
		{"let x = a - b;", true, "Binary subtraction"},
		{"let x = a * b;", true, "Binary multiplication"},
		{"let x = a / b;", true, "Binary division"},
		{"let x = -a;", true, "Unary negation"},
		{"let x = !a;", true, "Unary not"},
		{"let x = a && b;", true, "Logical AND"},
		{"let x = a || b;", true, "Logical OR"},
		{"let x = (a + b) * c;", true, "Parenthesized expression"},
		{"let x = a > b;", true, "Greater than comparison"},
		{"let x = a < b;", true, "Less than comparison"},
		{"let x = a >= b;", true, "Greater than or equal comparison"},
		{"let x = a <= b;", true, "Less than or equal comparison"},
		{"let x = a == b;", true, "Equality comparison"},
		{"let x = a != b;", true, "Inequality comparison"},
		{"let x = a + ;", false, "Missing right operand"},
		{"let x = * b;", false, "Missing left operand"},
		{"let x = (a + b;", false, "Unclosed parenthesis"},
		{"let x = a + + b;", false, "Invalid consecutive operators"},
		{"let x = a++;", true, "Postfix increment"},
		{"let x = a--;", true, "Postfix decrement"},
		{"let x = ++a;", true, "Prefix increment"},
		{"let x = --a;", true, "Prefix decrement"},
		{"let x = ++++a;", false, "Invalid consecutive prefix increment"},
		{"let x = ----a;", false, "Invalid consecutive prefix decrement"},
		{"let x = a++++b;", false, "Invalid consecutive postfix increment"},
		{"let x = a----b;", false, "Invalid consecutive postfix decrement"},
		{"let x = ++a++;", false, "Invalid mix of prefix and postfix increment"},
		{"let x = --a--;", false, "Invalid mix of prefix and postfix decrement"},
		{"let x = ++;", false, "Missing operand for prefix increment"},
		{"let x = --;", false, "Missing operand for prefix decrement"},
		{"let x = arr[0];", true, "Simple array indexing"},
		{"let x = arr[i + 1];", true, "Array indexing with expression"},
		{"let x = arr[arr[i]];", true, "Nested array indexing"},
		{"let x = arr[i][j];", true, "Multiple array indexing"},
		{"let x = arr[];", false, "Missing index expression"},
		{"let x = arr[;", false, "Unclosed array index"},
		{"let x = arr[1 + ];", false, "Invalid index expression"},
		{"arr[0] = 42;", true, "Array element assignment"},
		{"arr[i + 1] = x;", true, "Array assignment with expression index"},
		{"arr[0][1] = 42;", true, "Nested array element assignment"},
		{"return a + b;", true, "Return statement with expression"},
		{"return;", true, "Return statement with no expression"},
		{"return a, b;", true, "Return statement with multiple expressions"},
		{"return;", true, "Return statement with no expression"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}

func TestOnlyExpression(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		// expressions with no use case
		{"a;", true, "Single expression"},
		{"a, b;", true, "Multiple expressions"},
		{"a, b", false, "Expected semicolon"},
		{"a b", false, "Unexpected token b"},

		// binary expressions
		{"a + b;", true, "Binary addition"},
		{"a - b;", true, "Binary subtraction"},
		{"a * b;", true, "Binary multiplication"},
		{"a / b;", true, "Binary division"},
		{"a % b;", true, "Binary modulo"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
