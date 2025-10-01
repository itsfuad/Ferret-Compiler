package resolver

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/symbol"
	"compiler/report"
	"fmt"
)

func getImportKeys(imports map[string]*symbol.SymbolTable) []string {
	keys := make([]string, 0, len(imports))
	for k := range imports {
		keys = append(keys, k)
	}
	return keys
}

func resolveImportStmt(r *analyzer.AnalyzerNode, imp *ast.ImportStmt, cm *modules.Module) {
	if imp.ImportPath.Value == "" {
		return
	}

	if r.Debug {
		colors.YELLOW.Printf("resolving import %q as %q in module %q\n", imp.ImportPath.Value, imp.Alias, cm.AST.Alias)
	}

	// Resolve the import path based on context
	// For local imports within remote modules, convert to full GitHub path
	moduleKey := imp.ImportPath.Value

	//module must be parses and stored already
	module, err := r.Ctx.GetModule(moduleKey)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, imp.Loc(), "failed to get imported module: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	// collect functions from the imported module
	anz := analyzer.NewAnalyzerNode(module.AST, r.Ctx, r.Debug)
	ResolveProgram(anz)
	cm.SymbolTable.Imports[imp.Alias] = module.SymbolTable

	if r.Debug {
		colors.GREEN.Printf("successfully stored import %q in module %q\n", imp.Alias, cm.AST.Alias)
	}
}

func resolveImportedSymbol(r *analyzer.AnalyzerNode, res *ast.VarScopeResolution, cm *modules.Module) {

	if r.Debug {
		colors.CYAN.Printf("looking for module %q in imports of %q\n", res.Module.Name, cm.AST.Alias)
	}

	symbolTable, err := cm.SymbolTable.GetImportedModule(res.Module.Name)
	if err != nil {
		if r.Debug {
			colors.RED.Printf("available imports in %q: %v\n", cm.AST.Alias, getImportKeys(cm.SymbolTable.Imports))
		}
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), err.Error(), report.RESOLVER_PHASE)
		return
	}

	if _, found := symbolTable.Lookup(res.Identifier.Name); !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("symbol %q not found in module %q", res.Identifier.Name, res.Module.Name), report.RESOLVER_PHASE)
		return
	}

	// Mark the import as used when successfully resolving a symbol from it
	if cm.UsedImports == nil {
		cm.UsedImports = make(map[string]bool)
	}

	cm.UsedImports[res.Module.Name] = true

	if r.Debug {
		//print symbol X found in module Y imported from Z
		colors.TEAL.Printf("resolved imported symbol %q from module %q imported from %q\n", res.Identifier.Name, res.Module.Name, cm.AST.Alias)
	}
}
