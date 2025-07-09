package semantic

import (
	"compiler/internal/frontend/ast"
)

// SymbolKind represents the kind of symbol (variable, constant, function, type, etc.)
type SymbolKind int

const (
	SymbolVar SymbolKind = iota
	SymbolConst
	SymbolType // For built-in and user-defined types
	// SymbolFunc // Uncomment when adding function support
	// SymbolStruct // Uncomment when adding struct support
)

// Symbol represents a named entity in the program (variable, constant, type, etc.)
type Symbol struct {
	Name string
	Kind SymbolKind
	Type ast.DataType
}
