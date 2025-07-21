package resolver

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/ctx"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic/analyzer"
	"fmt"
)

func resolveImportStmt(r *analyzer.AnalyzerNode, imp *ast.ImportStmt, cm *ctx.Module) {
	if imp.ImportPath.Value == "" {
		r.Ctx.Reports.AddSyntaxError(r.Program.FullPath, imp.Loc(), "Import module name cannot be empty", report.COLLECTOR_PHASE)
		return
	}

	//module must be parses and stored already
	module, err := r.Ctx.GetModule(imp.ImportPath.Value)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, imp.Loc(), "Failed to get imported module: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}

	// collect functions from the imported module
	anz := analyzer.NewAnalyzerNode(module.AST, r.Ctx, r.Debug)
	ResolveProgram(anz)
	cm.SymbolTable.Imports[imp.ModuleName] = module.SymbolTable
}

func resolveImportedSymbol(r *analyzer.AnalyzerNode, res *ast.VarScopeResolution, cm *ctx.Module) {

	symbolTable, ok := cm.SymbolTable.Imports[res.Module.Name]
	if !ok {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Module '%s' is not imported", res.Module.Name), report.RESOLVER_PHASE)
		return
	}

	if _, found := symbolTable.Lookup(res.Identifier.Name); !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Symbol '%s' not found in module '%s'", res.Identifier.Name, res.Module.Name), report.RESOLVER_PHASE)
		return
	}

	if r.Debug {
		//print symbol X found in module Y imported from Z
		colors.TEAL.Printf("Resolved imported symbol '%s' from module '%s' imported from '%s'\n", res.Identifier.Name, res.Module.Name, cm.AST.Modulename)
	}
}
