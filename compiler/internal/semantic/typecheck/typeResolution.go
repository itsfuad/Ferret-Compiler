package typecheck

import (
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

// resolveTypeAlias resolves a type alias to its underlying type
func resolveTypeAlias(r *analyzer.AnalyzerNode, t semantic.Type) semantic.Type {
	userType, ok := t.(*semantic.UserType)
	if !ok {
		return t
	}

	// Try to find the type in current module first
	if resolvedType := resolveTypeInCurrentModule(r, userType); resolvedType != nil {
		return resolvedType
	}

	// If not found locally, check imported modules
	if resolvedType := resolveTypeInImportedModules(r, userType); resolvedType != nil {
		return resolvedType
	}

	// If not found, return the original type
	return t
}

// resolveTypeInCurrentModule tries to resolve a type in the current module
func resolveTypeInCurrentModule(r *analyzer.AnalyzerNode, userType *semantic.UserType) semantic.Type {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		return nil
	}

	sym, found := currentModule.SymbolTable.Lookup(string(userType.Name))
	if found && sym.Kind == semantic.SymbolType && sym.Type != nil {
		return sym.Type
	}
	return nil
}

// resolveTypeInImportedModules tries to resolve a type in imported modules
func resolveTypeInImportedModules(r *analyzer.AnalyzerNode, userType *semantic.UserType) semantic.Type {
	typeName := string(userType.Name)

	for _, moduleName := range r.Ctx.ModuleNames() {
		module, err := r.Ctx.GetModule(moduleName)
		if err != nil {
			continue
		}

		sym, found := module.SymbolTable.Lookup(typeName)
		if found && sym.Kind == semantic.SymbolType && sym.Type != nil {
			return sym.Type
		}
	}
	return nil
}
