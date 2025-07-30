package collector

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/semantic/analyzer"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/symbol"
	"fmt"
)

func CollectSymbols(c *analyzer.AnalyzerNode) {
	importPath := c.Program.ImportPath

	// Check if this module can be processed for collection phase
	if !c.Ctx.CanProcessPhase(importPath, modules.PHASE_COLLECTED) {
		currentPhase := c.Ctx.GetModulePhase(importPath)
		if currentPhase >= modules.PHASE_COLLECTED {
			// Already processed or in a later phase, skip
			if c.Debug {
				colors.BLUE.Printf("Skipping collection for '%s' (already in phase: %s)\n", c.Program.FullPath, currentPhase.String())
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
		colors.BLUE.Printf("Collected symbols for '%s'\n", c.Program.FullPath)
	}
}

func collectSymbols(c *analyzer.AnalyzerNode, node ast.Node, cm *modules.Module) {
	// collect functions for forward declarations
	switch n := node.(type) {
	case *ast.ImportStmt:
		collectSymbolsFromImport(c, n, cm)
	case *ast.FunctionDecl:
		collectFunctionSymbol(c, n, cm)
	}
}

func collectSymbolsFromImport(collector *analyzer.AnalyzerNode, imp *ast.ImportStmt, parentModule *modules.Module) {
	defer func() {
		if r := recover(); r != nil {
			collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), fmt.Sprintf("Panic while collecting symbols from import: %v", r), report.COLLECTOR_PHASE)
		}
	}()

	// Resolve the import path based on context
	// For local imports within remote modules, convert to full GitHub path
	moduleKey := modules.ResolveImportPath(imp.ImportPath.Value, collector.Program.FullPath, collector.Ctx.RemoteCachePath)

	// ✅ SECURITY CHECK: Validate remote import permissions
	if err := modules.CheckCanImportRemoteModules(collector.Ctx.ProjectRoot, moduleKey); err != nil {
		collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), err.Error(), report.COLLECTOR_PHASE)
		return
	}

	//module must be parses and stored already
	module, err := collector.Ctx.GetModule(moduleKey)
	if err != nil {
		collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), fmt.Sprintf("Failed to get imported module: %s", err.Error()), report.COLLECTOR_PHASE)
		return
	}

	//if already analyzed don't analyze again
	if module.Phase >= modules.PHASE_COLLECTED {
		return
	}

	// Collect symbols from the imported module recursively
	CollectSymbols(&analyzer.AnalyzerNode{
		Ctx:     collector.Ctx,
		Program: module.AST,
	})
}

func collectFunctionSymbol(c *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) {
	if fn.Identifier.Name == "" {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, fn.Loc(), "Function identifier cannot be empty", report.COLLECTOR_PHASE)
		return
	}

	// declare the function symbol
	symbol := symbol.NewSymbolWithLocation(fn.Identifier.Name, symbol.SymbolFunc, nil, fn.Loc())
	err := cm.SymbolTable.Declare(fn.Identifier.Name, symbol)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, fn.Loc(), "Failed to declare function symbol: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}
	if c.Debug {
		colors.GREEN.Printf("Declared function symbol '%s' at %s\n", fn.Identifier.Name, fn.Loc().String())
	}
}
