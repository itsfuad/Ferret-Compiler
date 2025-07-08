package semantic

import (
	"fmt"
)

// SymbolTable manages scoped symbols (variables, constants, etc.)
type SymbolTable struct {
	Symbols map[string]*Symbol
	Parent  *SymbolTable
	Imports map[string]*SymbolTable // alias -> imported module's symbol table
}

func NewSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Symbols: make(map[string]*Symbol),
		Parent:  parent,
		Imports: make(map[string]*SymbolTable),
	}
}

func (st *SymbolTable) Declare(name string, sym *Symbol) error {
	if _, exists := st.Symbols[name]; exists {
		return fmt.Errorf("symbol '%s' already declared in this scope", name)
	}
	st.Symbols[name] = sym
	return nil
}

func (st *SymbolTable) Lookup(name string) (*Symbol, bool) {
	if sym, ok := st.Symbols[name]; ok {
		return sym, true
	}
	if st.Parent != nil {
		return st.Parent.Lookup(name)
	}
	return nil, false
}
