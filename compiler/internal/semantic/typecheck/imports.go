package typecheck

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/stype"
	"compiler/report"
	"fmt"
)

func checkImportStmt(c *analyzer.AnalyzerNode, imp *ast.ImportStmt, cm *modules.Module) {
	if imp.ImportPath.Value == "" {
		return
	}

	// Resolve the import path based on context
	// For local imports within remote modules, convert to full GitHub path
	moduleKey := imp.ImportPath.Value

	//module must be parses and stored already
	module, err := c.Ctx.GetModule(moduleKey)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, imp.Loc(), "Failed to get imported module: "+err.Error(), report.TYPECHECK_PHASE)
		return
	}

	// process the imported module
	anz := analyzer.NewAnalyzerNode(module.AST, c.Ctx, c.Debug)
	CheckProgram(anz)
	cm.SymbolTable.Imports[imp.Alias] = module.SymbolTable
}

func checkImportedSymbolType(r *analyzer.AnalyzerNode, res *ast.VarScopeResolution, cm *modules.Module) stype.Type {

	symbolTable, err := cm.SymbolTable.GetImportedModule(res.Module.Name)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), err.Error(), report.RESOLVER_PHASE)
		return nil
	}

	resIdentifier, found := symbolTable.Lookup(res.Identifier.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Symbol %q not found in module %q", res.Identifier.Name, res.Module.Name), report.RESOLVER_PHASE)
		return nil
	}
	if resIdentifier.Type == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Symbol %q has no type defined", res.Identifier.Name), report.RESOLVER_PHASE)
		return nil
	}
	if r.Debug {
		//print symbol X found in module Y imported from Z
		colors.AQUA.Printf("Type Checked imported symbol %q of type %q from module %q imported from %q\n", res.Identifier.Name, resIdentifier.Type, res.Module.Name, cm.AST.Alias)
	}

	return resIdentifier.Type
}
