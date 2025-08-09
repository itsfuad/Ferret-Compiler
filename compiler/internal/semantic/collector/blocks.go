package collector

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/semantic/analyzer"
)

// collectSymbolsFromBlock collects symbols from all nodes in a block
func collectSymbolsFromBlock(c *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module) {
	if block == nil {
		return
	}

	for _, node := range block.Nodes {
		colors.BROWN.Printf("Collecting symbols from node <%T> at %s\n", node, node.Loc())
		collectSymbols(c, node, cm)
	}
}
