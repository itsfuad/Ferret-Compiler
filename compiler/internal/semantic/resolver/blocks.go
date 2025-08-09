package resolver

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/semantic/analyzer"
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
