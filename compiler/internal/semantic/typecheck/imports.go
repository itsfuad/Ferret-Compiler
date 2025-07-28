package typecheck

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/registry"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/stype"
	"fmt"
)

func checkImportStmt(c *analyzer.AnalyzerNode, imp *ast.ImportStmt, cm *ctx.Module) {
	if imp.ImportPath.Value == "" {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, imp.Loc(), "Import module name cannot be empty", report.TYPECHECK_PHASE)
		return
	}

	// Resolve the import path based on context
	// For local imports within remote modules, convert to full GitHub path
	moduleKey := registry.ResolveImportPath(imp.ImportPath.Value, c.Program.FullPath, c.Ctx)

	// ✅ SECURITY CHECK: Validate remote import permissions
	if err := registry.CheckCanImportRemoteModules(c.Ctx, moduleKey); err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, imp.Loc(), err.Error(), report.TYPECHECK_PHASE)
		return
	}

	//module must be parses and stored already
	module, err := c.Ctx.GetModule(moduleKey)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, imp.Loc(), "Failed to get imported module: "+err.Error(), report.TYPECHECK_PHASE)
		return
	}

	// process the imported module
	anz := analyzer.NewAnalyzerNode(module.AST, c.Ctx, c.Debug)
	CheckProgram(anz)
	cm.SymbolTable.Imports[imp.ModuleName] = module.SymbolTable
}

func checkImportedSymbolType(r *analyzer.AnalyzerNode, res *ast.VarScopeResolution, cm *ctx.Module) stype.Type {

	symbolTable, ok := cm.SymbolTable.Imports[res.Module.Name]
	if !ok {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Module '%s' is not imported", res.Module.Name), report.RESOLVER_PHASE)
		return nil
	}

	resIdentifier, found := symbolTable.Lookup(res.Identifier.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Symbol '%s' not found in module '%s'", res.Identifier.Name, res.Module.Name), report.RESOLVER_PHASE)
		return nil
	}
	if resIdentifier.Type == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Symbol '%s' has no type defined", res.Identifier.Name), report.RESOLVER_PHASE)
		return nil
	}
	if r.Debug {
		//print symbol X found in module Y imported from Z
		colors.AQUA.Printf("Resolved imported symbol '%s' of type '%s' from module '%s' imported from '%s'\n", res.Identifier.Name, resIdentifier.Type, res.Module.Name, cm.AST.Modulename)
	}

	return resIdentifier.Type
}
