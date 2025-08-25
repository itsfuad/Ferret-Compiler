package resolver

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/semantic/analyzer"
)

// resolveBlock resolves all nodes in a block
func resolveBlock(r *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module) {
	if block == nil {
		return
	}

	for _, node := range block.Nodes {
		resolveNode(r, node, cm)
	}
}
