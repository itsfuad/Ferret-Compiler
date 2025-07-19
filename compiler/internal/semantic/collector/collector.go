package collector

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)

func CollectSymbols(c *analyzer.AnalyzerNode) {
	currentModule, err := c.Ctx.GetModule(c.Program.ImportPath)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, c.Program.Loc(), "Failed to get current module: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}

	for _, node := range c.Program.Nodes {
		collectSymbols(c, node, currentModule)
	}

	if c.Debug {
		colors.BLUE.Printf("Collected symbols for '%s'\n", c.Program.FullPath)
	}
}

func collectSymbols(c *analyzer.AnalyzerNode, node ast.Node, cm *ctx.Module) {
	// collect functions for forward declarations
	switch n := node.(type) {
	case *ast.ImportStmt:
		collectSymbolsFromImport(c, n, cm)
	case *ast.FunctionDecl:
		collectFunctionSymbol(c, n, cm)
	}
}

func collectSymbolsFromImport(c *analyzer.AnalyzerNode, imp *ast.ImportStmt, cm *ctx.Module) {
	if imp.ImportPath.Value == "" {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, imp.Loc(), "Import module name cannot be empty", report.COLLECTOR_PHASE)
		return
	}

	//module must be parses and stored already
	module, err := c.Ctx.GetModule(imp.ImportPath.Value)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, imp.Loc(), "Failed to get imported module: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}

	// collect functions from the imported module
	anz := analyzer.NewAnalyzerNode(module.AST, c.Ctx, c.Debug)
	CollectSymbols(anz)
	cm.SymbolTable.Imports[imp.ModuleName] = module.SymbolTable
}

func collectFunctionSymbol(c *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *ctx.Module) {
	if fn.Identifier.Name == "" {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, fn.Loc(), "Function identifier cannot be empty", report.COLLECTOR_PHASE)
		return
	}

	// declare the function symbol
	symbol := ctx.NewSymbolWithLocation(fn.Identifier.Name, ctx.SymbolFunc, nil, fn.Loc())
	err := cm.SymbolTable.Declare(fn.Identifier.Name, symbol)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, fn.Loc(), "Failed to declare function symbol: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}
	if c.Debug {
		colors.GREEN.Printf("Declared function symbol '%s' at %s\n", fn.Identifier.Name, fn.Loc().String())
	}
}
