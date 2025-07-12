package typecheck

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

// checkAssignment performs type checking on assignments
func checkAssignment(r *analyzer.AnalyzerNode, stmt *ast.AssignmentStmt) {
	// Check each assignment pair
	leftExprs := *stmt.Left
	rightExprs := *stmt.Right

	for i, leftExpr := range leftExprs {
		if i >= len(rightExprs) {
			break // Mismatched assignment count - should be caught elsewhere
		}

		leftType := inferExpressionType(r, leftExpr)
		rightType := inferExpressionType(r, rightExprs[i])

		if leftType != nil && rightType != nil {
			if !semantic.IsAssignableFrom(leftType, rightType) {
				r.Ctx.Reports.Add(
					r.Program.FullPath,
					leftExpr.Loc(),
					"type mismatch: cannot assign "+rightType.String()+" to "+leftType.String(),
					report.TYPECHECK_PHASE,
				).SetLevel(report.SEMANTIC_ERROR)
			}
		}
	}
}

// checkExpressionStmt performs type checking on expression statements
func checkExpressionStmt(r *analyzer.AnalyzerNode, stmt *ast.ExpressionStmt) {
	if stmt.Expressions != nil {
		for _, expr := range *stmt.Expressions {
			inferExpressionType(r, expr) // This will catch type errors in expressions
		}
	}
}