package collector

import (
	"ferret/colors"
	"ferret/internal/frontend/ast"
	"ferret/internal/modules"
	"ferret/internal/semantic/analyzer"
	"ferret/report"
)

func CollectSymbols(c *analyzer.AnalyzerNode) {
	importPath := c.Program.ImportPath

	// Check if this module can be processed for collection phase
	if !c.Ctx.CanProcessPhase(importPath, modules.PHASE_COLLECTED) {
		currentPhase := c.Ctx.GetModulePhase(importPath)
		if currentPhase >= modules.PHASE_COLLECTED {
			// Already processed or in a later phase, skip
			if c.Debug {
				colors.BLUE.Printf("Skipping collection for %q (already in phase: %s)\n", c.Program.FullPath, currentPhase)
			}
			return
		}
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, c.Program.Loc(), "Module not ready for symbol collection phase", report.COLLECTOR_PHASE)
		return
	}

	currentModule, err := c.Ctx.GetModule(importPath)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, c.Program.Loc(), "Failed to get current module: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}

	for _, node := range c.Program.Nodes {
		collectSymbols(c, node, currentModule)
	}

	// Mark module as collected
	c.Ctx.SetModulePhase(importPath, modules.PHASE_COLLECTED)

	if c.Debug {
		colors.BLUE.Printf("Collected symbols for %q\n", c.Program.FullPath)
	}
}

func collectSymbols(c *analyzer.AnalyzerNode, node ast.Node, cm *modules.Module) {
	if node == nil {
		return
	}
	// collect functions for forward declarations
	switch n := node.(type) {
	case *ast.ImportStmt:
		collectSymbolsFromImport(c, n)
	case *ast.FunctionDecl:
		collectFunctionSymbol(c, n, cm)
	case *ast.MethodDecl:
		collectMethodSymbol(c, n, cm)
	case *ast.FunctionLiteral:
		collectFunctionLiteral(c, n, cm)
	case *ast.VarDeclStmt:
		collectVariableSymbols(c, n, cm)
	case *ast.TypeDeclStmt:
		collectTypeSymbol(c, n, cm)
	case *ast.IfStmt:
		collectSymbolsFromIfStmt(c, n, cm)
	case *ast.Block:
		collectSymbolsFromBlock(c, n, cm)
	case *ast.FunctionCallExpr:
		collectCall(c, n, cm)
	case *ast.ExpressionStmt:
		collectExprStmt(c, n, cm)
	case *ast.ReturnStmt:
		collectSymbols(c, *n.Value, cm)
	case *ast.BinaryExpr:
		// Recursively collect from both operands
		if n.Left != nil {
			collectSymbols(c, *n.Left, cm)
		}
		if n.Right != nil {
			collectSymbols(c, *n.Right, cm)
		}
	case *ast.PrefixExpr:
		// Recursively collect from the operand
		if n.Operand != nil {
			collectSymbols(c, *n.Operand, cm)
		}
	case *ast.PostfixExpr:
		// Recursively collect from the operand
		if n.Operand != nil {
			collectSymbols(c, *n.Operand, cm)
		}
		// For other expressions and nodes, we don't need to collect symbols
		// (literals, identifiers, etc. don't contain nested function literals)
	}
}

func collectCall(c *analyzer.AnalyzerNode, callExpr *ast.FunctionCallExpr, cm *modules.Module) {
	// Recursively collect from the caller (might be a function literal)
	if callExpr.Caller != nil {
		collectSymbols(c, *callExpr.Caller, cm)
	}
	// Recursively collect from arguments (might contain function literals)
	for _, arg := range callExpr.Arguments {
		collectSymbols(c, arg, cm)
	}
}

func collectExprStmt(c *analyzer.AnalyzerNode, exprStmt *ast.ExpressionStmt, cm *modules.Module) {
	if exprStmt.Expressions != nil {
		for _, expr := range *exprStmt.Expressions {
			collectSymbols(c, expr, cm)
		}
	}
}
