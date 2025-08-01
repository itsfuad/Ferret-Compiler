package typecheck

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"fmt"
)

// checkIfStmt validates an if statement and its branches
func checkIfStmt(r *analyzer.AnalyzerNode, ifStmt *ast.IfStmt, cm *modules.Module) {
	// 1. Check condition is boolean
	checkIfCondition(r, ifStmt.Condition, cm)

	// 2. Check the main body
	checkBlock(r, ifStmt.Body, cm)

	// 3. Check alternative (else/else-if)
	if ifStmt.Alternative != nil {
		checkAlternative(r, ifStmt.Alternative, cm)
	}
}

// checkIfCondition ensures the condition evaluates to a boolean type
func checkIfCondition(r *analyzer.AnalyzerNode, condition *ast.Expression, cm *modules.Module) {
	if condition == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			nil,
			"If statement missing condition",
			report.TYPECHECK_PHASE,
		)
		return
	}

	conditionType := evaluateExpressionType(r, *condition, cm)
	if conditionType == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			(*condition).Loc(),
			"Cannot determine type of if condition",
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Ensure condition is boolean
	if !semantic.IsBoolType(conditionType) {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			(*condition).Loc(),
			fmt.Sprintf("If condition must be boolean, got '%s'", conditionType.String()),
			report.TYPECHECK_PHASE,
		)
	}
}

// checkBlock validates all nodes in a block
func checkBlock(r *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module) {
	if block == nil {
		return
	}

	for _, node := range block.Nodes {
		checkNode(r, node, cm)
	}
}

// checkAlternative handles else and else-if branches
func checkAlternative(r *analyzer.AnalyzerNode, alternative ast.Node, cm *modules.Module) {
	switch alt := alternative.(type) {
	case *ast.IfStmt:
		// This is an "else if" - recursively check it
		checkIfStmt(r, alt, cm)
	case *ast.Block:
		// This is an "else" block
		checkBlock(r, alt, cm)
	default:
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			alternative.Loc(),
			fmt.Sprintf("Invalid alternative in if statement: %T", alternative),
			report.TYPECHECK_PHASE,
		)
	}
}
