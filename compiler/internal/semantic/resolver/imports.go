package resolver

import (
	"ferret/colors"
	"ferret/internal/frontend/ast"
	"ferret/internal/modules"
	"ferret/internal/semantic/analyzer"
	"ferret/internal/symbol"
	"ferret/report"
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
		r.Ctx.Reports.AddSyntaxError(r.Program.FullPath, imp.Loc(), "Import module name cannot be empty", report.RESOLVER_PHASE)
		return
	}

	if r.Debug {
		colors.YELLOW.Printf("Resolving import %q as %q in module %q\n", imp.ImportPath.Value, imp.ModuleName, cm.AST.Modulename)
	}

	// Resolve the import path based on context
	// For local imports within remote modules, convert to full GitHub path
	moduleKey := modules.ResolveImportPath(imp.ImportPath.Value, r.Program.FullPath, r.Ctx.RemoteCachePath)

	// ✅ SECURITY CHECK: Validate remote import permissions
	if err := modules.CheckCanImportRemoteModules(r.Ctx.ProjectRootFullPath, moduleKey); err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, imp.Loc(), err.Error(), report.RESOLVER_PHASE)
		return
	}

	//module must be parses and stored already
	module, err := r.Ctx.GetModule(moduleKey)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, imp.Loc(), "Failed to get imported module: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	// collect functions from the imported module
	anz := analyzer.NewAnalyzerNode(module.AST, r.Ctx, r.Debug)
	ResolveProgram(anz)
	cm.SymbolTable.Imports[imp.ModuleName] = module.SymbolTable

	if r.Debug {
		colors.GREEN.Printf("Successfully stored import %q in module %q\n", imp.ModuleName, cm.AST.Modulename)
	}
}

func resolveImportedSymbol(r *analyzer.AnalyzerNode, res *ast.VarScopeResolution, cm *modules.Module) {

	if r.Debug {
		colors.CYAN.Printf("Looking for module %q in imports of %q\n", res.Module.Name, cm.AST.Modulename)
	}

	symbolTable, err := cm.SymbolTable.GetImportedModule(res.Module.Name)
	if err != nil {
		if r.Debug {
			colors.RED.Printf("Available imports in %q: %v\n", cm.AST.Modulename, getImportKeys(cm.SymbolTable.Imports))
		}
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), err.Error(), report.RESOLVER_PHASE)
		return
	}

	if _, found := symbolTable.Lookup(res.Identifier.Name); !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Symbol %q not found in module %q", res.Identifier.Name, res.Module.Name), report.RESOLVER_PHASE)
		return
	}

	// Mark the import as used when successfully resolving a symbol from it
	if cm.UsedImports == nil {
		cm.UsedImports = make(map[string]bool)
	}

	cm.UsedImports[res.Module.Name] = true

	if r.Debug {
		//print symbol X found in module Y imported from Z
		colors.TEAL.Printf("Resolved imported symbol %q from module %q imported from %q\n", res.Identifier.Name, res.Module.Name, cm.AST.Modulename)
	}
}
