package collector

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic/analyzer"

	"ferret/compiler/internal/symbol"
	"fmt"
)

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
