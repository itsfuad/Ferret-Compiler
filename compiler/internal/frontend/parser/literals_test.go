package parser

import (
	"testing"
)

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
