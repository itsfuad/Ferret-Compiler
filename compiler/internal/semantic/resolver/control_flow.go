package resolver

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/semantic/analyzer"
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

// resolveBlock resolves all nodes in a block
func resolveBlock(r *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module) {
	if block == nil {
		return
	}

	for _, node := range block.Nodes {
		resolveNode(r, node, cm)
	}
}

// resolveReturnStmt resolves a return statement
func resolveReturnStmt(r *analyzer.AnalyzerNode, returnStmt *ast.ReturnStmt, cm *modules.Module) {
	// Resolve the return value expression if present
	if returnStmt.Value != nil {
		resolveExpr(r, *returnStmt.Value, cm)
	}
}
