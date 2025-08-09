package typecheck

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic"
	"ferret/compiler/internal/semantic/analyzer"
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/source"
	"ferret/compiler/internal/utils/msg"
	"fmt"
)

// ControlFlowResult represents the result of control flow analysis
type ControlFlowResult struct {
	AllPathsReturn         bool
	HasFallbackReturn      bool              // Function body has a return after conditionals
	CriticalMissingReturns []source.Location // Only critical missing returns
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
			fmt.Sprintf("If condition must be boolean, got '%s'", conditionType),
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
				fmt.Sprintf("Function must return a value of type '%s'", expectedReturnType),
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
	if ok, err := isImplicitCastable(expectedReturnType, returnValueType); !ok {
		rp := r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			returnStmt.Loc(),
			fmt.Sprintf("Cannot return '%s' in function expecting '%s': %s",
				returnValueType, expectedReturnType, err.Error()),
			report.TYPECHECK_PHASE,
		)
		if ok, _ := isExplicitCastable(expectedReturnType, returnValueType); ok {
			rp.AddHint(msg.CastHint(expectedReturnType))
		}
	}
}

// analyzeControlFlow performs comprehensive control flow analysis for a block
func analyzeControlFlow(r *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module, expectedReturnType stype.Type) ControlFlowResult {
	if block == nil {
		return createEmptyControlFlowResult()
	}

	result := createEmptyControlFlowResult()
	conditionalResults := []ControlFlowResult{}
	hasConditionals := false

	// Process all nodes in the block
	earlyReturn, hasConditionals := processBlockNodes(r, block, cm, expectedReturnType, &result, &conditionalResults, hasConditionals)
	if earlyReturn {
		return result
	}

	// Handle fallback return detection and missing returns
	return handleFallbackAndMissingReturns(block, result, conditionalResults, hasConditionals)
}

// createEmptyControlFlowResult creates a new empty ControlFlowResult
func createEmptyControlFlowResult() ControlFlowResult {
	return ControlFlowResult{
		AllPathsReturn:         false,
		HasFallbackReturn:      false,
		CriticalMissingReturns: []source.Location{},
	}
}

// processBlockNodes processes all nodes in a block and returns early if return found
func processBlockNodes(r *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module, expectedReturnType stype.Type,
	result *ControlFlowResult, conditionalResults *[]ControlFlowResult, hasConditionals bool) (bool, bool) {

	reachable := true

	for _, node := range block.Nodes {
		if !reachable {
			reportUnreachableCode(r, node)
			continue
		}

		switch n := node.(type) {
		case *ast.ReturnStmt:
			checkReturnStmt(r, n, cm, expectedReturnType)
			result.AllPathsReturn = true
			if !hasConditionals {
				result.HasFallbackReturn = true
			}
			return true, hasConditionals // Early return found
		case *ast.IfStmt:
			hasConditionals = true
			ifResult := analyzeIfStatement(r, n, cm, expectedReturnType)
			*conditionalResults = append(*conditionalResults, ifResult)
			if ifResult.AllPathsReturn {
				result.AllPathsReturn = true
				return true, hasConditionals // All paths in if statement return
			}
		default:
			checkNode(r, node, cm)
		}
	}

	return false, hasConditionals
}

// reportUnreachableCode reports unreachable code after return
func reportUnreachableCode(r *analyzer.AnalyzerNode, node ast.Node) {
	r.Ctx.Reports.AddSemanticError(
		r.Program.FullPath,
		node.Loc(),
		"Unreachable code after return statement",
		report.TYPECHECK_PHASE,
	)
}

// handleFallbackAndMissingReturns handles fallback return detection and collects missing returns
func handleFallbackAndMissingReturns(block *ast.Block, result ControlFlowResult, conditionalResults []ControlFlowResult, hasConditionals bool) ControlFlowResult {
	// Check for fallback return after conditionals
	if hasConditionals && hasFallbackReturn(block) {
		result.HasFallbackReturn = true
		result.AllPathsReturn = true
		return result
	}

	// Collect critical missing returns if no fallback
	if !result.HasFallbackReturn {
		collectMissingReturns(&result, conditionalResults, hasConditionals, block)
	}

	return result
}

// hasFallbackReturn checks if block has a return statement at the end
func hasFallbackReturn(block *ast.Block) bool {
	for i := len(block.Nodes) - 1; i >= 0; i-- {
		if _, ok := block.Nodes[i].(*ast.ReturnStmt); ok {
			return true
		}
	}
	return false
}

// collectMissingReturns collects all critical missing return locations
func collectMissingReturns(result *ControlFlowResult, conditionalResults []ControlFlowResult, hasConditionals bool, block *ast.Block) {
	for _, condResult := range conditionalResults {
		if !condResult.AllPathsReturn {
			result.CriticalMissingReturns = append(result.CriticalMissingReturns, condResult.CriticalMissingReturns...)
		}
	}

	// If no conditionals and no return, add end of block
	if !hasConditionals {
		if endLoc := getBlockEndLocation(block); endLoc != nil {
			result.CriticalMissingReturns = append(result.CriticalMissingReturns, *endLoc)
		}
	}
}

// analyzeIfStatement analyzes if statement for return paths
func analyzeIfStatement(r *analyzer.AnalyzerNode, ifStmt *ast.IfStmt, cm *modules.Module, expectedReturnType stype.Type) ControlFlowResult {
	// Check condition first
	checkIfCondition(r, ifStmt.Condition, cm)

	// Analyze main body
	mainResult := analyzeControlFlow(r, ifStmt.Body, cm, expectedReturnType)

	// Handle if-else vs if-only cases
	if ifStmt.Alternative != nil {
		return analyzeIfElseBranches(r, ifStmt, cm, expectedReturnType, mainResult)
	} else {
		return analyzeIfOnlyBranch(mainResult)
	}
}

// analyzeIfElseBranches handles if-else and if-else-if cases
func analyzeIfElseBranches(r *analyzer.AnalyzerNode, ifStmt *ast.IfStmt, cm *modules.Module, expectedReturnType stype.Type, mainResult ControlFlowResult) ControlFlowResult {
	result := createEmptyControlFlowResult()

	// Analyze alternative branch
	altResult := analyzeAlternativeBranch(r, ifStmt.Alternative, cm, expectedReturnType)

	// Both paths have returns only if BOTH main and alternative have returns
	if mainResult.AllPathsReturn && altResult.AllPathsReturn {
		result.AllPathsReturn = true
	} else {
		// Collect missing returns from problematic branches
		collectBranchMissingReturns(&result, mainResult, altResult)
	}

	return result
}

// analyzeIfOnlyBranch handles if statement without else
func analyzeIfOnlyBranch(mainResult ControlFlowResult) ControlFlowResult {
	result := createEmptyControlFlowResult()

	// If statement without else - execution can fall through
	result.AllPathsReturn = false
	if !mainResult.AllPathsReturn {
		result.CriticalMissingReturns = append(result.CriticalMissingReturns, mainResult.CriticalMissingReturns...)
	}

	return result
}

// analyzeAlternativeBranch analyzes else or else-if branch
func analyzeAlternativeBranch(r *analyzer.AnalyzerNode, alternative ast.Node, cm *modules.Module, expectedReturnType stype.Type) ControlFlowResult {
	switch alt := alternative.(type) {
	case *ast.IfStmt:
		return analyzeIfStatement(r, alt, cm, expectedReturnType)
	case *ast.Block:
		return analyzeControlFlow(r, alt, cm, expectedReturnType)
	default:
		return createEmptyControlFlowResult()
	}
}

// collectBranchMissingReturns collects missing returns from if-else branches
func collectBranchMissingReturns(result *ControlFlowResult, mainResult, altResult ControlFlowResult) {
	// Only report specific branches that are problematic
	if !mainResult.AllPathsReturn {
		result.CriticalMissingReturns = append(result.CriticalMissingReturns, mainResult.CriticalMissingReturns...)
	}
	if !altResult.AllPathsReturn {
		result.CriticalMissingReturns = append(result.CriticalMissingReturns, altResult.CriticalMissingReturns...)
	}
}

// getBlockEndLocation returns the location at the end of a block
func getBlockEndLocation(block *ast.Block) *source.Location {
	if block == nil || len(block.Nodes) == 0 {
		return nil
	}
	// Return the location of the last node in the block
	lastNode := block.Nodes[len(block.Nodes)-1]
	return lastNode.Loc()
}

// isVoidType checks if a type represents void (no return type)
func isVoidType(t stype.Type) bool {
	// In Ferret, void functions have nil return type
	return t == nil
}
