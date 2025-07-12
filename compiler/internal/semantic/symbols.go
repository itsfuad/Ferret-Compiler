package semantic

import (
	"compiler/internal/source"
)

// SymbolKind represents the kind of symbol (variable, constant, function, type, etc.)
type SymbolKind string

const (
	SymbolVar    SymbolKind = "variable"
	SymbolConst  SymbolKind = "constant"
	SymbolType   SymbolKind = "type" // For primitive and user-defined types
	SymbolFunc   SymbolKind = "function"
	SymbolStruct SymbolKind = "struct" // For struct types
	SymbolField  SymbolKind = "field"  // For struct fields
)

// Symbol represents a named entity in the program (variable, constant, type, etc.)
type Symbol struct {
	Name     string
	Kind     SymbolKind
	Type     Type             // Now uses semantic.Type instead of ast.DataType
	Location *source.Location // Optional, only when needed for error reporting
}

// NewSymbol creates a new symbol with the given properties
func NewSymbol(name string, kind SymbolKind, semanticType Type) *Symbol {
	return &Symbol{
		Name: name,
		Kind: kind,
		Type: semanticType,
	}
}

// NewSymbolWithLocation creates a new symbol with location information
func NewSymbolWithLocation(name string, kind SymbolKind, semanticType Type, loc *source.Location) *Symbol {
	return &Symbol{
		Name:     name,
		Kind:     kind,
		Type:     semanticType,
		Location: loc,
	}
}
