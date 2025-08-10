package resolver

import (
	"ferret/internal/frontend/ast"
	"ferret/internal/modules"
	"ferret/internal/semantic/analyzer"
)

// resolveIfStmt resolves an if statement and its branches
func resolveIfStmt(r *analyzer.AnalyzerNode, ifStmt *ast.IfStmt, cm *modules.Module) {
	// Resolve condition expression
	if ifStmt.Condition != nil {
		resolveExpr(r, *ifStmt.Condition, cm)
	}

	// Resolve body block
	if ifStmt.Body != nil {
		resolveBlock(r, ifStmt.Body, cm)
	}

	// Resolve alternative (else/else-if)
	if ifStmt.Alternative != nil {
		resolveNode(r, ifStmt.Alternative, cm)
	}
}

// resolveReturnStmt resolves a return statement
func resolveReturnStmt(r *analyzer.AnalyzerNode, returnStmt *ast.ReturnStmt, cm *modules.Module) {
	// Resolve the return value expression if present
	if returnStmt.Value != nil {
		resolveExpr(r, *returnStmt.Value, cm)
	}
}
