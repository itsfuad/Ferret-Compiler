package resolver

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

func resolveAssignment(r *analyzer.AnalyzerNode, stmt *ast.AssignmentStmt) { // Check that all left-hand side variables are declared
	for _, lhs := range *stmt.Left {
		if id, ok := lhs.(*ast.IdentifierExpr); ok {
			varSym, found := r.Ctx.Modules[r.Program.FullPath].SymbolTable.Lookup(id.Name)
			if !found {
				r.Ctx.Reports.Add(r.Program.FullPath, id.Loc(), "assignment to undeclared variable: "+id.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			} else if varSym.Type != nil {
				// Type checking: ensure type exists for variable
				typeName := string(varSym.Type.TypeName())
				typeSym, found := r.Ctx.Modules[r.Program.FullPath].SymbolTable.Lookup(typeName)
				if !found || typeSym.Kind != semantic.SymbolType {
					r.Ctx.Reports.Add(r.Program.FullPath, id.Loc(), "unknown type for variable: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
				}
			}
		} else {
			resolveExpr(r, lhs)
		}
	}
	// Check right-hand side expressions
	for _, rhs := range *stmt.Right {
		resolveExpr(r, rhs)
	}
}

func resolveExpressionStmt(r *analyzer.AnalyzerNode, stmt *ast.ExpressionStmt) {
	if stmt.Expressions != nil {
		for _, expr := range *stmt.Expressions {
			resolveExpr(r, expr)
		}
	}
}
