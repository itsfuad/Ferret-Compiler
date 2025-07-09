package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/test_helpers"
	"compiler/internal/types"
	"testing"
)

func assertIntType(t *testing.T, result ast.DataType, expected types.TYPE_NAME, bitSize uint8, unsigned bool) {
	t.Helper()
	intType, ok := result.(*ast.IntType)
	if !ok {
		t.Fatalf("expected *ast.IntType, got %T", result)
	}
	if intType.TypeName != expected {
		t.Errorf(test_helpers.ErrMsgFmt, "TypeName", intType.TypeName, expected)
	}
	if intType.BitSize != bitSize {
		t.Errorf(test_helpers.ErrMsgFmt, "BitSize", intType.BitSize, bitSize)
	}
	if intType.IsUnsigned != unsigned {
		t.Errorf(test_helpers.ErrMsgFmt, "Unsigned", intType.IsUnsigned, unsigned)
	}
}

func assertFloatType(t *testing.T, result ast.DataType, expected types.TYPE_NAME, bitSize uint8) {
	t.Helper()
	floatType, ok := result.(*ast.FloatType)
	if !ok {
		t.Fatalf("expected *ast.FloatType, got %T", result)
	}
	if floatType.TypeName != expected {
		t.Errorf(test_helpers.ErrMsgFmt, "TypeName", floatType.TypeName, expected)
	}
	if floatType.BitSize != bitSize {
		t.Errorf(test_helpers.ErrMsgFmt, "BitSize", floatType.BitSize, bitSize)
	}
}

func assertType(t *testing.T, result ast.DataType, expected types.TYPE_NAME, input string) {
	t.Helper()
	if result == nil {
		t.Fatalf("parseType() returned nil for input %s", input)
	}
	if result.Type() != expected {
		t.Errorf(test_helpers.ErrMsgFmt, "Type()", result.Type(), expected)
	}
}

func TestParseIntegerType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected types.TYPE_NAME
		bitSize  uint8
		unsigned bool
	}{
		{"i8", "i8", types.INT8, 8, false},
		{"i16", "i16", types.INT16, 16, false},
		{"i32", "i32", types.INT32, 32, false},
		{"i64", "i64", types.INT64, 64, false},
		{"u8", "u8", types.UINT8, 8, true},
		{"u16", "u16", types.UINT16, 16, true},
		{"u32", "u32", types.UINT32, 32, true},
		{"u64", "u64", types.UINT64, 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := test_helpers.CreateTestFileWithContent(t, tt.input)
			p := &Parser{
				tokens:      lexer.Tokenize(filePath, false),
				tokenNo:     0,
				filePathAbs: filePath,
			}
			p.tokens = []lexer.Token{
				{Kind: lexer.IDENTIFIER_TOKEN, Value: tt.input},
				{Kind: lexer.EOF_TOKEN},
			}

			result, ok := parseIntegerType(p)
			if !ok {
				t.Fatalf("parseIntegerType() returned nil for input %s", tt.input)
			}
			assertIntType(t, result, tt.expected, tt.bitSize, tt.unsigned)
		})
	}
}

func TestParseFloatType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected types.TYPE_NAME
		bitSize  uint8
	}{
		{"f32", "f32", types.FLOAT32, 32},
		{"f64", "f64", types.FLOAT64, 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := test_helpers.CreateTestFileWithContent(t, tt.input)
			p := &Parser{
				tokens:      lexer.Tokenize(filePath, false),
				tokenNo:     0,
				filePathAbs: filePath,
			}
			p.tokens = []lexer.Token{
				{Kind: lexer.IDENTIFIER_TOKEN, Value: tt.input},
				{Kind: lexer.EOF_TOKEN},
			}

			result, ok := parseFloatType(p)
			if !ok {
				t.Fatalf("parseFloatType() returned nil for input %s", tt.input)
			}
			assertFloatType(t, result, tt.expected, tt.bitSize)
		})
	}
}

func TestParseArrayType(t *testing.T) {

	tests := []struct {
		input    string
		isValid  bool
		desc     string
		expected types.TYPE_NAME
	}{
		{
			desc:     "Basic integer array type",
			input:    "type Array []i32;",
			isValid:  true,
			expected: types.ARRAY,
		},
		{
			desc:     "Basic float array type",
			input:    "type Array []f64;",
			isValid:  true,
			expected: types.ARRAY,
		},
		{
			desc:     "Basic string array type",
			input:    "type Array []str;",
			isValid:  true,
			expected: types.ARRAY,
		},
		{
			desc:     "Basic boolean array type",
			input:    "type Array []bool;",
			isValid:  true,
			expected: types.ARRAY,
		},
		{
			desc:     "Basic object array type",
			input:    "type Array []User;",
			isValid:  true,
			expected: types.ARRAY,
		},
		{
			desc:     "Invalid array",
			input:    "type Array []",
			isValid:  false,
			expected: types.ARRAY,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}

func TestParseType(t *testing.T) {
	tests := []struct {
		input    string
		isValid  bool
		desc     string
		expected types.TYPE_NAME
	}{
		{
			desc:     "Basic integer type",
			input:    "i32",
			isValid:  true,
			expected: types.INT32,
		},
		{
			desc:     "Basic float type",
			input:    "f64",
			isValid:  true,
			expected: types.FLOAT64,
		},
		{
			desc:     "String type",
			input:    "str",
			isValid:  true,
			expected: types.STRING,
		},
		{
			desc:     "Boolean type",
			input:    "bool",
			isValid:  true,
			expected: types.BOOL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			filePath := test_helpers.CreateTestFileWithContent(t, tt.input)
			p := &Parser{
				tokens:      lexer.Tokenize(filePath, false),
				tokenNo:     0,
				filePathAbs: filePath,
			}
			result, ok := parseType(p)
			if !tt.isValid && ok {
				t.Errorf("Expected nil result for invalid input %s", tt.input)
				return
			}

			if tt.isValid {
				assertType(t, result, tt.expected, tt.input)
			}
		})
	}
}

func TestTypeDeclaration(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		{"type Integer i32;", true, "Basic type declaration"},
		{"type String str;", true, "String type declaration"},
		{"type Numbers []i32;", true, "Array type declaration"},
		{"type;", false, "Missing type name"},
		{"type Integer;", false, "Missing underlying type"},
		{"type Integer i32", false, "Missing semicolon"},
		{"type 123 i32;", false, "Invalid type name"},
		{"type Integer [];", false, "Invalid underlying type"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}

func TestStructTypeDeclaration(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		// Valid cases
		{`type User struct { name: str, age: i32 };`, true, "Basic struct type declaration"},
		{`type User struct { name: str, age: i32, scores: []i32 };`, true, "Struct with array field"},
		{`type User struct { user: struct { name: str, age: i32 } };`, true, "Nested struct type"},
		{`type User struct { single: i32 };`, true, "Single field struct type"},
		{`type User struct { name: str };`, true, "Struct type without trailing comma"},

		// Invalid cases
		{`type User struct { };`, false, "Empty struct not allowed"},
		{`type User struct { name };`, false, "Missing field type"},
		{`type User struct { name: };`, false, "Missing type after colon"},
		{`type User struct { : str };`, false, "Missing field name"},
		{`type User struct { name: str age: i32 };`, false, "Missing comma between fields"},
		{`type User struct { name: str, name: i32 };`, false, "Duplicate field names"},
		{`type User struct { "name": str };`, false, "Non-identifier field name"},
		{`type User struct { name: str, age: i32 }`, false, "Missing semicolon"},
		{`type User struct { name: str, };`, true, "Struct type with trailing comma. But shows warning"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}

func TestInterfaceTypeDeclaration(t *testing.T) {
	tests := []struct {
		input   string
		isValid bool
		desc    string
	}{
		{`type INode interface {};`, true, "Basic interface type declaration with no methods"},
		{`type Shape interface { fn show() };`, true, "Interface with method declaration"},
		{`type INode interface { fn area() -> f64 };`, true, "Interface with method declaration and return type"},
		{`type INode interface { fn dostuff() -> (i32, str) };`, true, "Interface with method declaration and multiple return types"},

		//params
		{`type INode interface { fn dostuff(a: i32, b: str) -> i32 };`, true, "Interface with method declaration and parameters"},
		{`type INode interface { fn dostuff(a: i32, b: str) -> (i32, str) };`, true, "Interface with method declaration and parameters and return types"},

		//invalid
		{`type INode interface { name: str };`, false, "Interface with invalid field"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			testParseWithPanic(t, tt.input, tt.desc, tt.isValid)
		})
	}
}
