package semantic

import (
	"compiler/internal/types"
	"testing"
)

func TestAddPreludeSymbols(t *testing.T) {
	table := NewSymbolTable(nil)
	table = AddPreludeSymbols(table)

	testPreludeConstants(t, table)
	testPreludeTypes(t, table)
}

func testPreludeConstants(t *testing.T, table *SymbolTable) {
	testCases := []struct {
		name         string
		expectedKind SymbolKind
		expectedType types.TYPE_NAME
	}{
		{"true", SymbolConst, types.BOOL},
		{"false", SymbolConst, types.BOOL},
		{"null", SymbolConst, types.NULL},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateSymbol(t, table, tc.name, tc.expectedKind, tc.expectedType)
		})
	}
}

func testPreludeTypes(t *testing.T, table *SymbolTable) {
	typeTestCases := []struct {
		name         string
		expectedKind SymbolKind
		expectedType types.TYPE_NAME
	}{
		{"bool", SymbolType, types.BOOL},
		{"str", SymbolType, types.STRING},
		{"i32", SymbolType, types.INT32},
	}

	for _, tc := range typeTestCases {
		t.Run("type_"+tc.name, func(t *testing.T) {
			validateSymbol(t, table, tc.name, tc.expectedKind, tc.expectedType)
		})
	}
}

func validateSymbol(t *testing.T, table *SymbolTable, name string, expectedKind SymbolKind, expectedType types.TYPE_NAME) {
	symbol, found := table.Lookup(name)
	if !found {
		t.Fatalf("Symbol '%s' not found in symbol table", name)
	}

	if symbol.Kind != expectedKind {
		t.Errorf("Expected symbol kind %v for '%s', got %v", expectedKind, name, symbol.Kind)
	}

	if symbol.Type == nil {
		t.Fatalf("Symbol '%s' has nil type", name)
	}

	if symbol.Type.TypeName() != expectedType {
		t.Errorf("Expected type %v for '%s', got %v", expectedType, name, symbol.Type.TypeName())
	}
}

func TestBuiltInVariablesAreConstants(t *testing.T) {
	table := NewSymbolTable(nil)
	table = AddPreludeSymbols(table)

	constants := []string{"true", "false", "null"}

	for _, constName := range constants {
		symbol, found := table.Lookup(constName)
		if !found {
			t.Fatalf("Built-in constant '%s' not found", constName)
		}

		if symbol.Kind != SymbolConst {
			t.Errorf("Expected '%s' to be a constant (SymbolConst), got %v", constName, symbol.Kind)
		}
	}
}
