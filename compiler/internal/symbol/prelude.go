package symbol

import (
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/types"
)

func AddPreludeSymbols(table *SymbolTable) *SymbolTable {
	// Add primitive type symbols using semantic types

	// Integer types
	table.Declare("i8", NewSymbol("i8", SymbolType, &stype.PrimitiveType{TypeName: types.INT8}))
	table.Declare("i16", NewSymbol("i16", SymbolType, &stype.PrimitiveType{TypeName: types.INT16}))
	table.Declare("i32", NewSymbol("i32", SymbolType, &stype.PrimitiveType{TypeName: types.INT32}))
	table.Declare("i64", NewSymbol("i64", SymbolType, &stype.PrimitiveType{TypeName: types.INT64}))
	// Unsigned integer types
	table.Declare("u8", NewSymbol("u8", SymbolType, &stype.PrimitiveType{TypeName: types.UINT8}))
	table.Declare("u16", NewSymbol("u16", SymbolType, &stype.PrimitiveType{TypeName: types.UINT16}))
	table.Declare("u32", NewSymbol("u32", SymbolType, &stype.PrimitiveType{TypeName: types.UINT32}))
	table.Declare("u64", NewSymbol("u64", SymbolType, &stype.PrimitiveType{TypeName: types.UINT64}))
	// Floating point types
	table.Declare("f32", NewSymbol("f32", SymbolType, &stype.PrimitiveType{TypeName: types.FLOAT32}))
	table.Declare("f64", NewSymbol("f64", SymbolType, &stype.PrimitiveType{TypeName: types.FLOAT64}))
	// String type
	table.Declare("str", NewSymbol("str", SymbolType, &stype.PrimitiveType{TypeName: types.STRING}))
	// Boolean type
	table.Declare("bool", NewSymbol("bool", SymbolType, &stype.PrimitiveType{TypeName: types.BOOL}))
	// Byte type (Same as uint8)
	table.Declare("byte", NewSymbol("byte", SymbolType, &stype.PrimitiveType{TypeName: types.BYTE}))

	// Add boolean literals
	table.Declare("true", NewSymbol("true", SymbolVar, &stype.PrimitiveType{TypeName: types.BOOL}))
	table.Declare("false", NewSymbol("false", SymbolVar, &stype.PrimitiveType{TypeName: types.BOOL}))

	return table
}
