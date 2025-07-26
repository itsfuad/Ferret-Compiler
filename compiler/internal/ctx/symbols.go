package ctx

import (
	"compiler/internal/semantic/stype"
	"compiler/internal/source"
)

// SymbolKind represents the kind of symbol (variable, constant, function, type, etc.)
type SymbolKind int

const (
	SymbolVar SymbolKind = iota
	SymbolConst
	SymbolType   // For built-in and user-defined types
	SymbolFunc   // For functions
	SymbolStruct // For struct types
	SymbolField  // For struct fields
)

// Symbol represents a named entity in the program (variable, constant, type, etc.)
type Symbol struct {
	Name     string
	Kind     SymbolKind
	Type     stype.Type
	Location *source.Location
}

// NewSymbol creates a new symbol with the given properties
func NewSymbol(name string, kind SymbolKind, semanticType stype.Type) *Symbol {
	return &Symbol{
		Name: name,
		Kind: kind,
		Type: semanticType,
	}
}

// NewSymbolWithLocation creates a new symbol with location information
func NewSymbolWithLocation(name string, kind SymbolKind, semanticType stype.Type, loc *source.Location) *Symbol {
	return &Symbol{
		Name:     name,
		Kind:     kind,
		Type:     semanticType,
		Location: loc,
	}
}
