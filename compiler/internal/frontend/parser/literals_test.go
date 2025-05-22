package parser

import (
	"testing"
)

func TestLiteralParsing(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		// Integer literals
		{"let x = 42;", true, "Decimal integer literal"},
		{"let x = -42;", true, "Negative decimal integer literal"},
		{"let x = 1_000_000;", true, "Decimal integer with underscores"},
		{"let x = 0xFF;", true, "Hexadecimal literal"},
		{"let x = 0xff;", true, "Lowercase hexadecimal literal"},
		{"let x = 0xDEAD_BEEF;", true, "Hexadecimal with underscores"},
		{"let x = 0o777;", true, "Octal literal"},
		{"let x = 0o1_234_567;", true, "Octal with underscores"},
		{"let x = 0b1010;", true, "Binary literal"},
		{"let x = 0b1010_1010;", true, "Binary with underscores"},

		// Float literals
		{"let x = 3.14;", true, "Float literal"},
		{"let x = -3.14;", true, "Negative float literal"},
		{"let x = 1_234.567_89;", true, "Float with underscores"},
		{"let x = 1.2e-10;", true, "Scientific notation"},
		{"let x = 1.2E+10;", true, "Scientific notation with uppercase E"},
		{"let x = -1.2e-10;", true, "Negative scientific notation"},
		{"let x = 1_234.567_89e-10;", true, "Scientific notation with underscores"},

		// String literals
		{"let x = \"hello\";", true, "String literal"},
		{"let x = true;", true, "Boolean literal true"},
		{"let x = false;", true, "Boolean literal false"},

		// Array literals
		{"let x = [1, 2, 3];", true, "Array literal with numbers"},
		{"let x = [\"a\", \"b\"];", true, "Array literal with strings"},
		{"let x = [1, \"a\", true];", true, "Array literal with mixed types"},
		{"let x: []i32 = [1];", true, "Array literal with type annotation"},
		{"let x: []str = [\"hello\"];", true, "String array with type annotation"},

		// Invalid literals
		{"let x = \"unclosed;", false, "Unclosed string literal"},
		{"let x = 12.34.56;", false, "Invalid float literal"},
		{"let x = 0x;", false, "Invalid hex literal"},
		{"let x = 0xG;", false, "Invalid hex digit"},
		{"let x = 0o8;", false, "Invalid octal digit"},
		{"let x = 0o;", false, "Invalid octal literal"},
		{"let x = 0b;", false, "Invalid binary literal"},
		{"let x = 0b2;", false, "Invalid binary digit"},
		{"let x = 1.2e;", false, "Invalid scientific notation"},
		{"let x = [];", false, "Empty array literal not allowed"},
		{"let x = [1, 2;", false, "Unclosed array literal"},
		{"let x = [,];", false, "Invalid array literal with just comma"},
		{"let x = [1 2];", false, "Array literal missing comma"},
		{"let x = [1, 2,];", true, "Array literal with trailing comma"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}

func TestStructLiteralParsing(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		// Valid cases
		{`type Point struct { x: i32, y: i32 }; let x = @Point{x: 10, y: 20};`, true, "Basic struct literal"},
		{`type Point struct { x: i32, y: i32 }; let x = @Point{x: 10, y: 20, scores: [1, 2, 3]};`, true, "Struct with array field"},
		{`type User struct { name: str, age: i32 }; type Point struct { user: User, scores: []i32 }; let x = @Point{user: @User{name: "John", age: 20}, scores: [1, 2, 3]};`, true, "Nested struct literal"},
		{`let x = @struct{single: 42};`, true, "Single field struct literal"},
		{`let x = @struct{name: "John"};`, true, "Struct without trailing comma"},
		{`let x = @struct{name: "John", };`, true, "Struct literal with trailing comma"},
		//annonymous struct
		{`let x = @struct { name: str, age: i32 };`, true, "Anonymous struct literal"},

		// Invalid cases
		{`let x = @struct{};`, false, "Empty struct literal"},
		{`let x = @struct{name};`, false, "Missing field value"},
		{`let x = @struct{name: };`, false, "Missing value after colon"},
		{`let x = @struct{: "John"};`, false, "Missing field name"},
		{`let x = @struct{name: "John" age: 20};`, false, "Missing comma between fields"},
		{`let x = @struct{name: "John", name: "Jane"};`, false, "Duplicate field names"},
		{`let x = @struct{"name": "John"};`, false, "Non-identifier field name"},
		{`let x = @struct{name: "John", age: 20}`, false, "Missing semicolon"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
