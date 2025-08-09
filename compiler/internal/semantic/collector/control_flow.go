package collector

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/semantic/analyzer"
)

// collectSymbolsFromIfStmt collects symbols from an if statement and its branches
func collectSymbolsFromIfStmt(c *analyzer.AnalyzerNode, ifStmt *ast.IfStmt, cm *modules.Module) {
	// Collect symbols from the main body
	if ifStmt.Body != nil {
		collectSymbolsFromBlock(c, ifStmt.Body, cm)
	}

	// Collect symbols from alternative (else/else-if)
	if ifStmt.Alternative != nil {
		collectSymbols(c, ifStmt.Alternative, cm)
	}
}
