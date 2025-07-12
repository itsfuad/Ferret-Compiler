package semantic

import (
	"compiler/internal/types"
)

func AddPreludeSymbols(table *SymbolTable) *SymbolTable {
	// Add primitive type symbols using semantic types
	table.Declare("i8", NewSymbol("i8", SymbolType, CreatePrimitiveType(types.INT8)))
	table.Declare("i16", NewSymbol("i16", SymbolType, CreatePrimitiveType(types.INT16)))
	table.Declare("i32", NewSymbol("i32", SymbolType, CreatePrimitiveType(types.INT32)))
	table.Declare("i64", NewSymbol("i64", SymbolType, CreatePrimitiveType(types.INT64)))
	table.Declare("u8", NewSymbol("u8", SymbolType, CreatePrimitiveType(types.UINT8)))
	table.Declare("u16", NewSymbol("u16", SymbolType, CreatePrimitiveType(types.UINT16)))
	table.Declare("u32", NewSymbol("u32", SymbolType, CreatePrimitiveType(types.UINT32)))
	table.Declare("u64", NewSymbol("u64", SymbolType, CreatePrimitiveType(types.UINT64)))
	table.Declare("f32", NewSymbol("f32", SymbolType, CreatePrimitiveType(types.FLOAT32)))
	table.Declare("f64", NewSymbol("f64", SymbolType, CreatePrimitiveType(types.FLOAT64)))
	table.Declare("str", NewSymbol("str", SymbolType, CreatePrimitiveType(types.STRING)))
	table.Declare("bool", NewSymbol("bool", SymbolType, CreatePrimitiveType(types.BOOL)))
	table.Declare("byte", NewSymbol("byte", SymbolType, CreatePrimitiveType(types.BYTE)))
	return table
}
