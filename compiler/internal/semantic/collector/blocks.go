package collector

import (
	"ferret/colors"
	"ferret/internal/frontend/ast"
	"ferret/internal/modules"
	"ferret/internal/semantic/analyzer"
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
