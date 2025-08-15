package symbol

import (
	"testing"

	"compiler/internal/semantic/stype"
	"compiler/internal/types"
)

func TestAddPreludeSymbols(t *testing.T) {
	table := NewSymbolTable(nil)
	AddPreludeSymbols(table)

	tests := []struct {
		name     string
		wantType SymbolKind
		wantPrim types.TYPE_NAME
	}{
		{"i8", SymbolType, types.INT8},
		{"i16", SymbolType, types.INT16},
		{"i32", SymbolType, types.INT32},
		{"i64", SymbolType, types.INT64},
		{"u8", SymbolType, types.UINT8},
		{"u16", SymbolType, types.UINT16},
		{"u32", SymbolType, types.UINT32},
		{"u64", SymbolType, types.UINT64},
		{"f32", SymbolType, types.FLOAT32},
		{"f64", SymbolType, types.FLOAT64},
		{"str", SymbolType, types.STRING},
		{"bool", SymbolType, types.BOOL},
		{"byte", SymbolType, types.BYTE},
	}

	for _, tt := range tests {
		sym, ok := table.Lookup(tt.name)
		if !ok {
			t.Errorf("Symbol %q not found in table", tt.name)
			continue
		}
		if sym.Kind != tt.wantType {
			t.Errorf("Symbol %q: got kind %v, want %v", tt.name, sym.Kind, tt.wantType)
		}
		prim, ok := sym.Type.(*stype.PrimitiveType)
		if !ok {
			t.Errorf("Symbol %q: type is not *stype.PrimitiveType, got %T", tt.name, sym.Type)
			continue
		}
		if prim.TypeName != tt.wantPrim {
			t.Errorf("Symbol %q: got primitive name %v, want %v", tt.name, prim.TypeName, tt.wantPrim)
		}
	}
}

// Optionally, test that no extra symbols are added
func TestAddPreludeSymbolsNoExtraSymbols(t *testing.T) {
	table := NewSymbolTable(nil)

	AddPreludeSymbols(table)

	expected := map[string]struct{}{
		"i8": {}, "i16": {}, "i32": {}, "i64": {},
		"u8": {}, "u16": {}, "u32": {}, "u64": {},
		"f32": {}, "f64": {}, "str": {}, "bool": {}, "byte": {},
		"true": {}, "false": {},
	}

	for name := range table.Symbols {
		if _, ok := expected[name]; !ok {
			t.Errorf("Unexpected symbol in table: %q", name)
		}
	}
	if len(table.Symbols) != len(expected) {
		t.Errorf("Expected %d symbols, got %d", len(expected), len(table.Symbols))
	}
}
