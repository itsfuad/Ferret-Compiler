package cmd

import (
	"compiler/config"
	"compiler/internal/ctx"
	"compiler/internal/symbol"
)

// SymbolQueryAPI provides public access to compiler internals for symbol queries
type SymbolQueryAPI struct {
	Context *ctx.CompilerContext
}

// SymbolLocation represents the location of a symbol
type SymbolLocation struct {
	File   string
	Line   int
	Column int
}

// SymbolDetails contains detailed information about a symbol
type SymbolDetails struct {
	Name     string
	Kind     string
	Type     string
	Location *SymbolLocation
}

// GetContext returns the compiler context (for advanced use)
func (api *SymbolQueryAPI) GetContext() *ctx.CompilerContext {
	return api.Context
}

// GetBuiltinSymbols returns the builtin symbol table
func (api *SymbolQueryAPI) GetBuiltinSymbols() map[string]*SymbolDetails {
	if api.Context == nil || api.Context.Builtins == nil {
		return make(map[string]*SymbolDetails)
	}

	result := make(map[string]*SymbolDetails)
	for name, sym := range api.Context.Builtins.Symbols {
		result[name] = convertSymbol(sym, "builtin")
	}
	return result
}

// GetModuleSymbols returns symbols for a specific module
func (api *SymbolQueryAPI) GetModuleSymbols(modulePath string) map[string]*SymbolDetails {
	if api.Context == nil || api.Context.Modules == nil {
		return make(map[string]*SymbolDetails)
	}

	module, exists := api.Context.Modules[modulePath]
	if !exists {
		return make(map[string]*SymbolDetails)
	}

	result := make(map[string]*SymbolDetails)

	// Get the full file path from the module's AST
	fullPath := ""
	if module.AST != nil {
		fullPath = module.AST.FullPath
	}

	collectSymbolsRecursiveWithFile(module.SymbolTable, fullPath, result)
	return result
}

// GetAllModules returns a list of all module paths
func (api *SymbolQueryAPI) GetAllModules() []string {
	if api.Context == nil || api.Context.Modules == nil {
		return []string{}
	}

	modules := make([]string, 0, len(api.Context.Modules))
	for modulePath := range api.Context.Modules {
		modules = append(modules, modulePath)
	}
	return modules
}

// LookupSymbol searches for a symbol by name across all scopes
func (api *SymbolQueryAPI) LookupSymbol(name string) []*SymbolDetails {
	if api.Context == nil {
		return []*SymbolDetails{}
	}

	results := []*SymbolDetails{}

	// Check builtins
	if sym, found := api.Context.Builtins.Lookup(name); found {
		results = append(results, convertSymbol(sym, "builtin"))
	}

	// Check all modules
	for _, module := range api.Context.Modules {
		if sym, found := module.SymbolTable.Lookup(name); found {
			fullPath := ""
			if module.AST != nil {
				fullPath = module.AST.FullPath
			}
			results = append(results, convertSymbol(sym, fullPath))
		}
	}

	return results
}

// GetModuleInfo returns information about a specific module
func (api *SymbolQueryAPI) GetModuleInfo(modulePath string) map[string]interface{} {
	if api.Context == nil || api.Context.Modules == nil {
		return nil
	}

	module, exists := api.Context.Modules[modulePath]
	if !exists {
		return nil
	}

	// Get the full path from the AST if available
	fullPath := ""
	if module.AST != nil {
		fullPath = module.AST.FullPath
	}

	return map[string]interface{}{
		"full_path":    fullPath,
		"symbol_count": countSymbols(module.SymbolTable),
	}
}

// HasErrors returns whether compilation had errors
func (api *SymbolQueryAPI) HasErrors() bool {
	if api.Context == nil {
		return true
	}
	return api.Context.Reports.HasErrors()
}

// GetErrorCount returns the number of errors
func (api *SymbolQueryAPI) GetErrorCount() int {
	if api.Context == nil {
		return 0
	}
	return len(api.Context.Reports)
}

// GetStatistics returns compilation statistics
func (api *SymbolQueryAPI) GetStatistics() map[string]interface{} {
	if api.Context == nil {
		return map[string]interface{}{
			"modules":         0,
			"total_symbols":   0,
			"builtin_symbols": 0,
			"has_errors":      true,
			"error_count":     0,
		}
	}

	totalSymbols := len(api.Context.Builtins.Symbols)
	for _, module := range api.Context.Modules {
		totalSymbols += countSymbols(module.SymbolTable)
	}

	return map[string]interface{}{
		"modules":         len(api.Context.Modules),
		"total_symbols":   totalSymbols,
		"builtin_symbols": len(api.Context.Builtins.Symbols),
		"has_errors":      api.Context.Reports.HasErrors(),
		"error_count":     len(api.Context.Reports),
	}
}

// Helper functions

func convertSymbol(sym *symbol.Symbol, filePath string) *SymbolDetails {
	details := &SymbolDetails{
		Name: sym.Name,
		Kind: convertSymbolKind(sym.Kind),
	}

	if sym.Type != nil {
		details.Type = sym.Type.String()
	}

	if sym.Location != nil && sym.Location.Start != nil {
		details.Location = &SymbolLocation{
			File:   filePath,
			Line:   sym.Location.Start.Line,
			Column: sym.Location.Start.Column,
		}
	}

	return details
}

func convertSymbolKind(kind symbol.SymbolKind) string {
	switch kind {
	case symbol.SymbolVar:
		return "variable"
	case symbol.SymbolConst:
		return "constant"
	case symbol.SymbolType:
		return "type"
	case symbol.SymbolFunc:
		return "function"
	case symbol.SymbolMethod:
		return "method"
	case symbol.SymbolStruct:
		return "struct"
	case symbol.SymbolField:
		return "field"
	default:
		return "unknown"
	}
}

func collectSymbolsRecursiveWithFile(table *symbol.SymbolTable, filePath string, result map[string]*SymbolDetails) {
	if table == nil {
		return
	}

	for name, sym := range table.Symbols {
		result[name] = convertSymbol(sym, filePath)
	}

	// Note: We don't recurse into child scopes here, as that would include
	// local variables from function bodies. We only want top-level symbols.
	// If needed, this can be extended with a flag for recursive collection.
}

func countSymbols(table *symbol.SymbolTable) int {
	if table == nil {
		return 0
	}
	return len(table.Symbols)
}

// CompileForSymbolQuery compiles a project and returns the API for symbol queries
func CompileForSymbolQuery(projectRoot string, isDebug bool) (*SymbolQueryAPI, error) {
	// Load project config
	projectConfig, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	// Compile the project
	context := Compile(projectConfig, isDebug)

	return &SymbolQueryAPI{
		Context: context,
	}, nil
}
