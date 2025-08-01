package collector

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
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
		collectSymbolsFromImport(c, n)
	case *ast.FunctionDecl:
		collectFunctionSymbol(c, n, cm)
	case *ast.VarDeclStmt:
		collectVariableSymbols(c, n, cm)
	case *ast.TypeDeclStmt:
		collectTypeSymbol(c, n, cm)
	case *ast.IfStmt:
		collectSymbolsFromIfStmt(c, n, cm)
	case *ast.Block:
		collectSymbolsFromBlock(c, n, cm)
	}
}

func collectSymbolsFromImport(collector *analyzer.AnalyzerNode, imp *ast.ImportStmt) {
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
	if !declareFunctionSymbol(c, fn, cm) {
		return
	}

	functionScope := createFunctionScope(fn, cm)
	if functionScope == nil {
		return
	}

	collectFunctionParameters(c, fn, functionScope)
	collectFunctionBody(c, fn, cm, functionScope)
}

// declareFunctionSymbol declares the function symbol in the module's symbol table
func declareFunctionSymbol(c *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) bool {
	if fn.Identifier.Name == "" {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, fn.Loc(), "Function identifier cannot be empty", report.COLLECTOR_PHASE)
		return false
	}

	functionSymbol := symbol.NewSymbolWithLocation(fn.Identifier.Name, symbol.SymbolFunc, nil, fn.Loc())
	err := cm.SymbolTable.Declare(fn.Identifier.Name, functionSymbol)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, fn.Loc(), "Failed to declare function symbol: "+err.Error(), report.COLLECTOR_PHASE)
		return false
	}
	
	if c.Debug {
		colors.GREEN.Printf("Declared function symbol '%s' at %s\n", fn.Identifier.Name, fn.Loc().String())
	}
	return true
}

// createFunctionScope creates and stores a local symbol table for the function
func createFunctionScope(fn *ast.FunctionDecl, cm *modules.Module) *symbol.SymbolTable {
	functionScope := symbol.NewSymbolTable(cm.SymbolTable)
	cm.FunctionScopes[fn.Identifier.Name] = functionScope
	return functionScope
}

// collectFunctionParameters collects symbols from function parameters in the function's local scope
func collectFunctionParameters(c *analyzer.AnalyzerNode, fn *ast.FunctionDecl, functionScope *symbol.SymbolTable) {
	if fn.Function == nil {
		return
	}

	for _, param := range fn.Function.Params {
		if param.Identifier == nil {
			continue
		}
		
		paramSymbol := symbol.NewSymbolWithLocation(param.Identifier.Name, symbol.SymbolVar, nil, param.Identifier.Loc())
		paramErr := functionScope.Declare(param.Identifier.Name, paramSymbol)
		if paramErr != nil {
			c.Ctx.Reports.AddCriticalError(c.Program.FullPath, param.Identifier.Loc(), "Failed to declare parameter symbol: "+paramErr.Error(), report.COLLECTOR_PHASE)
			continue
		}
		
		if c.Debug {
			colors.GREEN.Printf("Declared parameter symbol '%s' (incomplete) at %s\n", param.Identifier.Name, param.Identifier.Loc().String())
		}
	}
}

// collectFunctionBody collects symbols from function body in the function's local scope
func collectFunctionBody(c *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module, functionScope *symbol.SymbolTable) {
	if fn.Function == nil || fn.Function.Body == nil {
		return
	}

	// Temporarily switch to function scope for body collection
	originalTable := cm.SymbolTable
	cm.SymbolTable = functionScope
	collectSymbolsFromBlock(c, fn.Function.Body, cm)
	cm.SymbolTable = originalTable // Restore module scope
}

func collectVariableSymbols(c *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *modules.Module) {
	for _, variable := range decl.Variables {
		if variable.Identifier.Name == "" {
			c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, variable.Identifier.Loc(), "Variable identifier cannot be empty", report.COLLECTOR_PHASE)
			continue
		}

		// Declare the variable symbol with placeholder type
		variableSymbol := symbol.NewSymbolWithLocation(variable.Identifier.Name, symbol.SymbolVar, nil, variable.Identifier.Loc())
		err := cm.SymbolTable.Declare(variable.Identifier.Name, variableSymbol)
		if err != nil {
			c.Ctx.Reports.AddCriticalError(c.Program.FullPath, variable.Identifier.Loc(), "Failed to declare variable symbol: "+err.Error(), report.COLLECTOR_PHASE)
			continue
		}
		if c.Debug {
			colors.GREEN.Printf("Declared variable symbol '%s' (incomplete) at %s\n", variable.Identifier.Name, variable.Identifier.Loc().String())
		}
	}
}

func collectTypeSymbol(c *analyzer.AnalyzerNode, decl *ast.TypeDeclStmt, cm *modules.Module) {
	aliasName := decl.Alias.Name
	if aliasName == "" {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, decl.Alias.Loc(), "Type alias name cannot be empty", report.COLLECTOR_PHASE)
		return
	}

	// Declare the type symbol with placeholder type
	typeSymbol := symbol.NewSymbolWithLocation(aliasName, symbol.SymbolType, nil, decl.Alias.Loc())
	err := cm.SymbolTable.Declare(aliasName, typeSymbol)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, decl.Alias.Loc(), "Failed to declare type symbol: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}
	if c.Debug {
		colors.GREEN.Printf("Declared type symbol '%s' (incomplete) at %s\n", aliasName, decl.Alias.Loc().String())
	}
}

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

// collectSymbolsFromBlock collects symbols from all nodes in a block
func collectSymbolsFromBlock(c *analyzer.AnalyzerNode, block *ast.Block, cm *modules.Module) {
	if block == nil {
		return
	}

	for _, node := range block.Nodes {
		collectSymbols(c, node, cm)
	}
}
