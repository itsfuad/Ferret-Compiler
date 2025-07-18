package semantic

import (
	"compiler/internal/types"
)

func AddPreludeSymbols(table *SymbolTable) *SymbolTable {
	// Add primitive type symbols using semantic types
	table.Declare("i8", NewSymbol("i8", SymbolType, &PrimitiveType{Name: types.INT8}))
	table.Declare("i16", NewSymbol("i16", SymbolType, &PrimitiveType{Name: types.INT16}))
	table.Declare("i32", NewSymbol("i32", SymbolType, &PrimitiveType{Name: types.INT32}))
	table.Declare("i64", NewSymbol("i64", SymbolType, &PrimitiveType{Name: types.INT64}))
	table.Declare("u8", NewSymbol("u8", SymbolType, &PrimitiveType{Name: types.UINT8}))
	table.Declare("u16", NewSymbol("u16", SymbolType, &PrimitiveType{Name: types.UINT16}))
	table.Declare("u32", NewSymbol("u32", SymbolType, &PrimitiveType{Name: types.UINT32}))
	table.Declare("u64", NewSymbol("u64", SymbolType, &PrimitiveType{Name: types.UINT64}))
	table.Declare("f32", NewSymbol("f32", SymbolType, &PrimitiveType{Name: types.FLOAT32}))
	table.Declare("f64", NewSymbol("f64", SymbolType, &PrimitiveType{Name: types.FLOAT64}))
	table.Declare("str", NewSymbol("str", SymbolType, &PrimitiveType{Name: types.STRING}))
	table.Declare("bool", NewSymbol("bool", SymbolType, &PrimitiveType{Name: types.BOOL}))
	table.Declare("byte", NewSymbol("byte", SymbolType, &PrimitiveType{Name: types.BYTE}))
	return table
}
