package semantic

import (
	"fmt"

	"compiler/internal/frontend/ast"
	"compiler/internal/source"
	"compiler/internal/types"
)

func getIntType(bitSize uint8) ast.DataType {
	return &ast.IntType{TypeName: types.TYPE_NAME(fmt.Sprintf("i%d", bitSize)), BitSize: bitSize, IsUnsigned: true, Location: source.Location{}}
}

func getUIntType(bitSize uint8) ast.DataType {
	return &ast.IntType{TypeName: types.TYPE_NAME(fmt.Sprintf("u%d", bitSize)), BitSize: bitSize, IsUnsigned: true, Location: source.Location{}}
}

func getFloatType(bitSize uint8) ast.DataType {
	return &ast.FloatType{TypeName: types.TYPE_NAME(fmt.Sprintf("f%d", bitSize)), BitSize: bitSize, Location: source.Location{}}
}

func getStringType() ast.DataType {
	return &ast.StringType{Location: source.Location{}}
}

func getBoolType() ast.DataType {
	return &ast.BoolType{Location: source.Location{}}
}

func getByteType() ast.DataType {
	return &ast.ByteType{Location: source.Location{}}
}

func AddPreludeSymbols(table *SymbolTable) *SymbolTable {
	table.Declare("i8", &Symbol{Name: "i8", Kind: SymbolType, Type: getIntType(8)})
	table.Declare("i16", &Symbol{Name: "i16", Kind: SymbolType, Type: getIntType(16)})
	table.Declare("i32", &Symbol{Name: "i32", Kind: SymbolType, Type: getIntType(32)})
	table.Declare("i64", &Symbol{Name: "i64", Kind: SymbolType, Type: getIntType(64)})
	table.Declare("u8", &Symbol{Name: "u8", Kind: SymbolType, Type: getUIntType(8)})
	table.Declare("u16", &Symbol{Name: "u16", Kind: SymbolType, Type: getUIntType(16)})
	table.Declare("u32", &Symbol{Name: "u32", Kind: SymbolType, Type: getUIntType(32)})
	table.Declare("u64", &Symbol{Name: "u64", Kind: SymbolType, Type: getUIntType(64)})
	table.Declare("f32", &Symbol{Name: "f32", Kind: SymbolType, Type: getFloatType(32)})
	table.Declare("f64", &Symbol{Name: "f64", Kind: SymbolType, Type: getFloatType(64)})
	table.Declare("str", &Symbol{Name: "str", Kind: SymbolType, Type: getStringType()})
	table.Declare("bool", &Symbol{Name: "bool", Kind: SymbolType, Type: getBoolType()})
	table.Declare("byte", &Symbol{Name: "byte", Kind: SymbolType, Type: getByteType()})
	return table
}
