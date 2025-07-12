package resolver

import (
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)


func resolveImport(r *analyzer.AnalyzerNode, currentModule *ctx.Module, importStmt *ast.ImportStmt) {
	if importStmt.ModuleName != "" && importStmt.FullPath != "" {
		importModule, err := r.Ctx.GetModule(importStmt.ImportPath.Value)
		if err == nil {
			importModuleAST := importModule.AST
			//semantic.AddPreludeSymbols(importModule.SymbolTable)
			anz := analyzer.NewAnalyzerNode(importModuleAST, r.Ctx, r.Debug)
			ResolveProgram(anz)
			currentModule.SymbolTable.Imports[importStmt.ModuleName] = importModule.SymbolTable
		} else {
			r.Ctx.Reports.Add(r.Program.FullPath, importStmt.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
	}
}
