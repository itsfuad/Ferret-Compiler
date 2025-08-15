package collector

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/semantic/analyzer"
)

// collectSymbolsFromBlock collects symbols from all nodes in a block
func collectSymbolsFromBlock(c *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module) {
	if block == nil {
		return
	}

	for _, node := range block.Nodes {
		collectSymbols(c, node, cm)
	}
}
