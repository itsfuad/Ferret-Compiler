package semantic

import (
	"fmt"
)

// SymbolTable manages scoped symbols (variables, constants, etc.)
type SymbolTable struct {
	parent  *SymbolTable
	symbols map[string]*Symbol
}

func NewSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		parent:  parent,
		symbols: make(map[string]*Symbol),
	}
}

func (st *SymbolTable) Declare(name string, sym *Symbol) error {
	if _, exists := st.symbols[name]; exists {
		return fmt.Errorf("symbol '%s' already declared in this scope", name)
	}
	st.symbols[name] = sym
	return nil
}

func (st *SymbolTable) Lookup(name string) (*Symbol, bool) {
	if sym, ok := st.symbols[name]; ok {
		return sym, true
	}
	if st.parent != nil {
		return st.parent.Lookup(name)
	}
	return nil, false
}
