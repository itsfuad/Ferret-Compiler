package resolver

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/semantic/analyzer"
)

func resolveBlockStmt(r *analyzer.AnalyzerNode, block *ast.Block) {
	if block == nil {
		return
	}
	// Resolve each statement in the block
	for _, node := range block.Nodes {
		resolveASTNode(r, node)
	}
}
