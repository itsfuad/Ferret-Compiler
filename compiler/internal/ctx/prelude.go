package ctx

import (
	atype "compiler/internal/types"
	"compiler/internal/semantic/types"
)

func AddPreludeSymbols(table *SymbolTable) *SymbolTable {
	// Add primitive type symbols using semantic types
	table.Declare("i8", NewSymbol("i8", SymbolType, &types.PrimitiveType{Name: atype.INT8}))
	table.Declare("i16", NewSymbol("i16", SymbolType, &types.PrimitiveType{Name: atype.INT16}))
	table.Declare("i32", NewSymbol("i32", SymbolType, &types.PrimitiveType{Name: atype.INT32}))
	table.Declare("i64", NewSymbol("i64", SymbolType, &types.PrimitiveType{Name: atype.INT64}))
	table.Declare("u8", NewSymbol("u8", SymbolType, &types.PrimitiveType{Name: atype.UINT8}))
	table.Declare("u16", NewSymbol("u16", SymbolType, &types.PrimitiveType{Name: atype.UINT16}))
	table.Declare("u32", NewSymbol("u32", SymbolType, &types.PrimitiveType{Name: atype.UINT32}))
	table.Declare("u64", NewSymbol("u64", SymbolType, &types.PrimitiveType{Name: atype.UINT64}))
	table.Declare("f32", NewSymbol("f32", SymbolType, &types.PrimitiveType{Name: atype.FLOAT32}))
	table.Declare("f64", NewSymbol("f64", SymbolType, &types.PrimitiveType{Name: atype.FLOAT64}))
	table.Declare("str", NewSymbol("str", SymbolType, &types.PrimitiveType{Name: atype.STRING}))
	table.Declare("bool", NewSymbol("bool", SymbolType, &types.PrimitiveType{Name: atype.BOOL}))
	table.Declare("byte", NewSymbol("byte", SymbolType, &types.PrimitiveType{Name: atype.BYTE}))
	return table
}
