package parser

import (
	"testing"
)

func TestStructParsing(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		// Field access
		{"point.x;", true, "Simple field access"},
		{"person.address.street;", true, "Chained field access"},
		{"point.;", false, "Missing field name"},
		{"point.123;", false, "Invalid field name"},
		{"point..x;", false, "Double dot operator"},

		// Struct types
		{"type Point struct { x: i32, y: i32 };", true, "Simple struct type"},
		{"let data: struct { name: str, address: struct { street: str, city: str } };", true, "Nested struct type"},
		{"type Point struct {};", false, "Empty struct type"},
		{"type Point struct { x: i32, x: i32 };", false, "Duplicate field names in struct type"},
		{"type Point struct { x: i32, };", true, "Trailing comma in struct type. Get warning"},
		{"type Point struct { ,x: i32 };", false, "Leading comma in struct type"},
		{"type Point struct { x: i32 y: i32 };", false, "Missing comma between struct fields"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
