package collector

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

func CollectSymbols(c *analyzer.AnalyzerNode) {

	for _, node := range c.Program.Nodes {
		collectSymbols(c, node)
	}

	if c.Debug {
		colors.BLUE.Printf("Collected symbols for '%s'\n", c.Program.FullPath)
	}
}

func collectSymbols(c *analyzer.AnalyzerNode, node ast.Node) {
	// collect functions for forward declarations
	switch n := node.(type) {
	case *ast.FunctionDecl:
		collectFunctionSymbol(c, n)
	}
}

func collectFunctionSymbol(c *analyzer.AnalyzerNode, fn *ast.FunctionDecl) {
	if fn.Identifier.Name == "" {
		c.Ctx.Reports.Add(c.Program.FullPath, fn.Loc(), "Function identifier cannot be empty", report.COLLECTOR_PHASE).SetLevel(report.SYNTAX_ERROR)
		return
	}

	currentModule, err := c.Ctx.GetModule(c.Program.ImportPath)
	if err != nil {
		c.Ctx.Reports.Add(c.Program.FullPath, fn.Loc(), "Failed to get current module: "+err.Error(), report.COLLECTOR_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	// declare the function symbol
	symbol := semantic.NewSymbolWithLocation(fn.Identifier.Name, semantic.SymbolFunc, nil, fn.Loc())
	err = currentModule.SymbolTable.Declare(fn.Identifier.Name, symbol)
	if err != nil {
		c.Ctx.Reports.Add(c.Program.FullPath, fn.Loc(), "Failed to declare function symbol: "+err.Error(), report.COLLECTOR_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}
	if c.Debug {
		colors.GREEN.Printf("Declared function symbol '%s' at %s\n", fn.Identifier.Name, fn.Loc().String())
	}
}
