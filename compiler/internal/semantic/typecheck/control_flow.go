package typecheck

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/stype"
	"fmt"
)

// ControlFlowResult represents the result of control flow analysis
type ControlFlowResult struct {
	HasReturn   bool       // Whether this path has a return statement
	IsReachable bool       // Whether code after this construct is reachable
	ReturnType  stype.Type // Type of return value (if any)
}

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

// checkReturnStmt validates a return statement
func checkReturnStmt(r *analyzer.AnalyzerNode, returnStmt *ast.ReturnStmt, cm *modules.Module, expectedReturnType stype.Type) {
	if returnStmt.Value == nil {
		// This is a bare return statement
		if expectedReturnType != nil && !isVoidType(expectedReturnType) {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				returnStmt.Loc(),
				fmt.Sprintf("Function must return a value of type '%s'", expectedReturnType.String()),
				report.TYPECHECK_PHASE,
			)
		}
		return
	}

	// This return has a value
	returnValueType := evaluateExpressionType(r, *returnStmt.Value, cm)
	if returnValueType == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			(*returnStmt.Value).Loc(),
			"Cannot determine type of return value",
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Check if function expects void
	if expectedReturnType == nil || isVoidType(expectedReturnType) {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			returnStmt.Loc(),
			"Void function cannot return a value",
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Check return type compatibility
	if !IsAssignableFrom(expectedReturnType, returnValueType) {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			returnStmt.Loc(),
			fmt.Sprintf("Cannot return '%s' in function expecting '%s'",
				returnValueType.String(), expectedReturnType.String()),
			report.TYPECHECK_PHASE,
		)
	}
}

// analyzeControlFlow performs comprehensive control flow analysis for a block
func analyzeControlFlow(r *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module, expectedReturnType stype.Type) ControlFlowResult {
	if block == nil {
		return ControlFlowResult{HasReturn: false, IsReachable: true}
	}

	result := ControlFlowResult{HasReturn: false, IsReachable: true}
	reachable := true

	for _, node := range block.Nodes {
		if !reachable {
			// Dead code detected
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				node.Loc(),
				"Unreachable code after return statement",
				report.TYPECHECK_PHASE,
			)
			continue
		}

		switch n := node.(type) {
		case *ast.ReturnStmt:
			checkReturnStmt(r, n, cm, expectedReturnType)
			result.HasReturn = true
			reachable = false // Code after return is unreachable
		case *ast.IfStmt:
			ifResult := analyzeIfStatement(r, n, cm, expectedReturnType)
			if ifResult.HasReturn {
				result.HasReturn = true
			}
			if !ifResult.IsReachable {
				reachable = false
			}
		default:
			// Check other nodes normally
			checkNode(r, node, cm)
		}
	}

	result.IsReachable = reachable
	return result
}

// analyzeIfStatement analyzes if statement for return paths
func analyzeIfStatement(r *analyzer.AnalyzerNode, ifStmt *ast.IfStmt, cm *modules.Module, expectedReturnType stype.Type) ControlFlowResult {
	// Check condition first
	checkIfCondition(r, ifStmt.Condition, cm)

	// Analyze main body
	mainResult := analyzeControlFlow(r, ifStmt.Body, cm, expectedReturnType)

	// Analyze alternative (else/else-if)
	var altResult ControlFlowResult
	if ifStmt.Alternative != nil {
		switch alt := ifStmt.Alternative.(type) {
		case *ast.IfStmt:
			altResult = analyzeIfStatement(r, alt, cm, expectedReturnType)
		case *ast.Block:
			altResult = analyzeControlFlow(r, alt, cm, expectedReturnType)
		}
	} else {
		// No else branch - code is reachable
		altResult = ControlFlowResult{HasReturn: false, IsReachable: true}
	}

	// Combine results
	// Both paths have returns only if BOTH main and alternative have returns
	hasReturn := mainResult.HasReturn && altResult.HasReturn
	// Code is reachable if either path is reachable
	isReachable := mainResult.IsReachable || altResult.IsReachable

	return ControlFlowResult{
		HasReturn:   hasReturn,
		IsReachable: isReachable,
	}
}

// isVoidType checks if a type represents void (no return type)
func isVoidType(t stype.Type) bool {
	// In Ferret, void functions have nil return type
	return t == nil
}
