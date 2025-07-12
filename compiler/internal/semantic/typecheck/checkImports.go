package typecheck

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)

// checkImportStmt performs type checking on import statements
func checkImportStmt(r *analyzer.AnalyzerNode, stmt *ast.ImportStmt) {
	//check the imported module
	importModule, err := r.Ctx.GetModule(stmt.ImportPath.Value)
	if err != nil {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			stmt.Loc(),
			err.Error(),
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	if importModule.AST == nil {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			stmt.Loc(),
			"imported module has no AST: "+stmt.ImportPath.Value,
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	//typecheck it
	anz := analyzer.NewAnalyzerNode(importModule.AST, r.Ctx, r.Debug)
	CheckProgram(anz)
}
