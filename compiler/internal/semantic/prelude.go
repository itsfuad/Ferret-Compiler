package semantic

import (
	"compiler/internal/types"
)

func AddPreludeSymbols(table *SymbolTable) *SymbolTable {
	table.Declare("i8", &Symbol{Name: "i8", Kind: SymbolType, Type: types.INT8})
	table.Declare("i16", &Symbol{Name: "i16", Kind: SymbolType, Type: types.INT16})
	table.Declare("i32", &Symbol{Name: "i32", Kind: SymbolType, Type: types.INT32})
	table.Declare("i64", &Symbol{Name: "i64", Kind: SymbolType, Type: types.INT64})
	table.Declare("u8", &Symbol{Name: "u8", Kind: SymbolType, Type: types.UINT8})
	table.Declare("u16", &Symbol{Name: "u16", Kind: SymbolType, Type: types.UINT16})
	table.Declare("u32", &Symbol{Name: "u32", Kind: SymbolType, Type: types.UINT32})
	table.Declare("u64", &Symbol{Name: "u64", Kind: SymbolType, Type: types.UINT64})
	table.Declare("f32", &Symbol{Name: "f32", Kind: SymbolType, Type: types.FLOAT32})
	table.Declare("f64", &Symbol{Name: "f64", Kind: SymbolType, Type: types.FLOAT64})
	table.Declare("str", &Symbol{Name: "str", Kind: SymbolType, Type: types.STRING})
	table.Declare("bool", &Symbol{Name: "bool", Kind: SymbolType, Type: types.BOOL})
	table.Declare("byte", &Symbol{Name: "byte", Kind: SymbolType, Type: types.BYTE})
	table.Declare("void", &Symbol{Name: "void", Kind: SymbolType, Type: types.VOID})
	return table
}
