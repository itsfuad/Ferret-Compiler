package symbol

import (
	"fmt"
)

// SymbolTable manages scoped symbols (variables, constants, etc.)
type SymbolTable struct {
	Symbols map[string]*Symbol
	Parent  *SymbolTable
	Imports map[string]*SymbolTable // alias -> imported module's symbol table
	// Track import paths to detect duplicate imports of same module
	ImportPaths map[string]string // alias -> import path
}

func NewSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Symbols:     make(map[string]*Symbol),
		Parent:      parent,
		Imports:     make(map[string]*SymbolTable),
		ImportPaths: make(map[string]string),
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

// AddImport adds an imported module to this symbol table
// Returns error if the alias already exists with a different import path
func (st *SymbolTable) AddImport(alias, importPath string, moduleSymbolTable *SymbolTable) error {
	// Check if this exact module (import path) is already imported
	for existingAlias, existingPath := range st.ImportPaths {
		if existingPath == importPath {
			if existingAlias == alias {
				return fmt.Errorf("'%s' already imported", importPath)
			} else {
				return fmt.Errorf("'%s' already imported with alias '%s'", importPath, existingAlias)
			}
		}
	}

	// Check if the alias is already used by a different module
	if existingPath, exists := st.ImportPaths[alias]; exists {
		if existingPath != importPath {
			return fmt.Errorf("alias '%s' is already used by import '%s'. Use a different alias with 'as'", alias, existingPath)
		}
	}

	st.Imports[alias] = moduleSymbolTable
	st.ImportPaths[alias] = importPath
	return nil
}

// CheckImportConflict checks if an alias would conflict with existing imports
func (st *SymbolTable) CheckImportConflict(alias string) (bool, string) {
	if existingPath, exists := st.ImportPaths[alias]; exists {
		return true, existingPath
	}
	return false, ""
}

// GetImportAliases returns all import aliases in this symbol table
func (st *SymbolTable) GetImportAliases() []string {
	aliases := make([]string, 0, len(st.Imports))
	for alias := range st.Imports {
		aliases = append(aliases, alias)
	}
	return aliases
}

// IsImportUsed checks if an import alias has been used (has any lookups)
func (st *SymbolTable) IsImportUsed(alias string) bool {
	// This is a simple implementation - in a more sophisticated system,
	// we'd track actual usage during symbol resolution
	if moduleTable, exists := st.Imports[alias]; exists {
		// For now, consider an import used if its symbol table exists
		// A more advanced implementation would track actual symbol lookups
		return moduleTable != nil
	}
	return false
}
