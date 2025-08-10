package resolver

import (
	"ferret/internal/frontend/ast"
	"ferret/internal/modules"
	"ferret/internal/semantic/analyzer"
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
