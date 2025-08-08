package collector

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic/analyzer"

	//"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/symbol"
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
				colors.BLUE.Printf("Skipping collection for '%s' (already in phase: %s)\n", c.Program.FullPath, currentPhase)
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
	if node == nil {
		return
	}

	colors.BROWN.Printf("Collecting symbols from node <%T> at %s\n", node, node.Loc())
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
		// Recursively collect from the caller (might be a function literal)
		if n.Caller != nil {
			collectSymbols(c, *n.Caller, cm)
		}
		// Recursively collect from arguments (might contain function literals)
		for _, arg := range n.Arguments {
			collectSymbols(c, arg, cm)
		}
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

func collectSymbolsFromImport(collector *analyzer.AnalyzerNode, imp *ast.ImportStmt) {
	defer func() {
		if r := recover(); r != nil {
			collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), fmt.Sprintf("Panic while collecting symbols from import: %v", r), report.COLLECTOR_PHASE)
		}
	}()

	colors.BLUE.Printf("Collecting symbols from import '%s' at %s\n", imp.ImportPath.Value, imp.Loc())

	// Get the current module
	currentModule, err := collector.Ctx.GetModule(collector.Ctx.FullPathToImportPath(collector.Program.FullPath))
	if err != nil {
		collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), "Failed to get current module for import validation", report.COLLECTOR_PHASE)
		return
	}

	// Resolve the import path based on context
	// For local imports within remote modules, convert to full GitHub path
	moduleKey := modules.ResolveImportPath(imp.ImportPath.Value, collector.Program.FullPath, collector.Ctx.RemoteCachePath)
	colors.BLUE.Sprintf("moduleKey: %s", moduleKey)

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

	// Add import to current module's symbol table with validation
	alias := imp.ModuleName
	if err := currentModule.SymbolTable.AddImport(alias, moduleKey, module.SymbolTable); err != nil {
		collector.Ctx.Reports.AddSemanticError(
			collector.Program.FullPath,
			imp.Loc(),
			err.Error(),
			report.COLLECTOR_PHASE,
		)
		return
	}

	if collector.Debug {
		colors.GREEN.Printf("Added import '%s' with alias '%s' to module '%s'\n", moduleKey, alias, collector.Ctx.FullPathToImportPath(collector.Program.FullPath))
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

	functionScope := declareFunctionSymbol(c, fn.Function, cm.SymbolTable, symbol.SymbolFunc)
	if functionScope == nil {
		return
	}

	collectFunctionParameters(c, fn.Function, functionScope)
	collectFunctionBody(c, fn.Function, cm, functionScope)
}

func collectFunctionLiteral(c *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, cm *modules.Module) {

	colors.AQUA.Printf("Collecting function literal '%s' at %s\n", fn.ID, fn.Loc())

	functionScope := declareFunctionSymbol(c, fn, cm.SymbolTable, symbol.SymbolFunc)
	if functionScope == nil {
		return
	}

	// Collect parameters and body in the function's local scope
	collectFunctionParameters(c, fn, functionScope)
	collectFunctionBody(c, fn, cm, functionScope)
}

// declareFunctionSymbol declares the function symbol in the module's symbol table
func declareFunctionSymbol(c *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, parentScope *symbol.SymbolTable, kind symbol.SymbolKind) *symbol.SymbolTable {
	if fn.ID == "" {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, fn.Loc(), "identifier cannot be empty", report.COLLECTOR_PHASE)
		return nil
	}

	functionSymbol := symbol.NewSymbolWithLocation(fn.ID, kind, nil, fn.Loc())
	err := parentScope.Declare(fn.ID, functionSymbol)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, fn.Loc(), "Failed to declare symbol: "+err.Error(), report.COLLECTOR_PHASE)
		return nil
	}

	functionSymbol.SelfScope.ScopeName = symbol.SYMBOL_TABLE_FUNCTION

	colors.GREEN.Printf("Declared function symbol '%s' (incomplete) at %s\n", fn.ID, fn.Loc())

	return functionSymbol.SelfScope // Return the function's local scope for further collection
}

// collectFunctionParameters collects symbols from function parameters in the function's local scope
func collectFunctionParameters(c *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, functionScope *symbol.SymbolTable) {
	if fn == nil {
		return
	}

	for _, param := range fn.Params {
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
			colors.GREEN.Printf("Declared parameter symbol '%s' (incomplete) at %s\n", param.Identifier.Name, param.Identifier.Loc())
		}
	}
}

// collectFunctionBody collects symbols from function body in the function's local scope
func collectFunctionBody(c *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, cm *modules.Module, functionScope *symbol.SymbolTable) {
	if fn == nil || fn.Body == nil {
		return
	}

	// Temporarily switch to function scope for body collection
	originalTable := cm.SymbolTable
	cm.SymbolTable = functionScope
	collectSymbolsFromBlock(c, fn.Body, cm)
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
			colors.GREEN.Printf("Declared variable symbol '%s' (incomplete) at %s\n", variable.Identifier.Name, variable.Identifier.Loc())
		}
	}
	// Collect initializers if any
	for _, initializer := range decl.Initializers {
		if initializer == nil {
			continue
		}
		collectSymbols(c, initializer, cm) // Collect symbols from the initializer expression
		if c.Debug {
			colors.GREEN.Printf("Collected symbols from initializer at %s\n", initializer.Loc())
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
		colors.GREEN.Printf("Declared type symbol '%s' (incomplete) at %s\n", aliasName, decl.Alias.Loc())
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
		colors.BROWN.Printf("Collecting symbols from node <%T> at %s\n", node, node.Loc())
		collectSymbols(c, node, cm)
	}
}

func collectMethodSymbol(c *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module) {
	
	//collect reciever
	if method.Receiver == nil {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, method.Loc(), "Method receiver cannot be nil", report.COLLECTOR_PHASE)
		return
	}

	//must be user-defined type
	utype, ok := method.Receiver.Type.(*ast.UserDefinedType)
	if !ok {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, method.Receiver.Identifier.Loc(), "Method receiver must be a user-defined type", report.COLLECTOR_PHASE)
		return
	}

	// Check if the receiver type is already defined
	receiverSymbol, found := cm.SymbolTable.Lookup(string(utype.TypeName))
	if !found {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, method.Receiver.Identifier.Loc(), fmt.Sprintf("Receiver type '%s' not found in symbol table", utype.TypeName), report.COLLECTOR_PHASE)
		return
	}

	if receiverSymbol.SelfScope == nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, method.Receiver.Identifier.Loc(), fmt.Sprintf("Receiver type '%s' does not have a valid symbol table", utype.TypeName), report.COLLECTOR_PHASE)
		return
	}

	// Declare the method symbol with placeholder type
	methodSymbol := symbol.NewSymbolWithLocation(method.Method.Name, symbol.SymbolMethod, nil, method.Method.Loc())
	
	// new scope for method
	methodScope := symbol.NewSymbolTable(receiverSymbol.SelfScope)
	methodScope.ScopeName = symbol.SYMBOL_TABLE_FUNCTION
	err := receiverSymbol.SelfScope.Declare(method.Method.Name, methodSymbol)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, method.Method.Loc(), "Failed to declare method symbol: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}

	// declare the receiver in the method scope
	receiverParamSymbol := symbol.NewSymbolWithLocation(method.Receiver.Identifier.Name, symbol.SymbolField, nil, method.Receiver.Identifier.Loc())
	err = methodSymbol.SelfScope.Declare(method.Receiver.Identifier.Name, receiverParamSymbol)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, method.Receiver.Identifier.Loc(), "Failed to declare receiver symbol: "+err.Error(), report.COLLECTOR_PHASE)
		return
	} else {
		colors.GREEN.Printf("Declared receiver symbol '%s' (incomplete) at %s\n", method.Receiver.Identifier.Name, method.Receiver.Identifier.Loc())
	}

	collectFunctionParameters(c, method.Function, methodScope)
	collectFunctionBody(c, method.Function, cm, methodScope)
}