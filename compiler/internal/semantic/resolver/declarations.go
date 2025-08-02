package resolver

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/stype"
	"compiler/internal/symbol"
	"compiler/internal/types"
	"fmt"
)

func resolveFunctionDecl(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) {

	functionSymbol, found := cm.SymbolTable.Lookup(fn.Identifier.Name)
	if !found {
		colors.RED.Printf("Function '%s' not found in symbol table\n", fn.Identifier.Name)
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function '"+fn.Identifier.Name+"' is not declared", report.RESOLVER_PHASE)
		return
	}

	// Get function scope from the function symbol itself
	functionScope := functionSymbol.Scope
	if functionScope == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function scope for '"+fn.Identifier.Name+"' not found", report.RESOLVER_PHASE)
		return
	}

	// Resolve parameter types and update function scope symbols
	paramTypes := resolveParameterTypes(r, fn.Function, cm, functionScope)
	if paramTypes == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to resolve parameter types for function '"+fn.Identifier.Name+"'", report.RESOLVER_PHASE)
		return // Error occurred during parameter resolution
	}

	// Resolve return type
	returnType := resolveReturnType(r, fn.Function, cm)
	if returnType == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to resolve return type for function '"+fn.Identifier.Name+"'", report.RESOLVER_PHASE)
		return // Error occurred during return type resolution
	}

	fmt.Printf("Resolved function '%s' with parameters: %v and return type: %v\n", fn.Identifier.Name, paramTypes, returnType)
	// Resolve function body
	//set scope
	originalTable := cm.SymbolTable
	cm.SymbolTable = functionScope
	resolveBlock(r, fn.Function.Body, cm)
	// Restore original module scope
	cm.SymbolTable = originalTable
	// Create and assign function type
	functionType := stype.FunctionType{
		Parameters: paramTypes,
		ReturnType: returnType,
	}
	functionSymbol.Type = &functionType
}

func resolveParameterTypes(r *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, cm *modules.Module, functionScope *symbol.SymbolTable) []stype.Type {
	paramTypes := make([]stype.Type, 0) // Initialize as empty slice, not nil slice

	// Check if function has no parameters
	if fn == nil || fn.Params == nil || len(fn.Params) == 0 {
		return paramTypes // Return empty slice for functions with no parameters
	}

	if functionScope == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function scope not found", report.RESOLVER_PHASE)
		return nil // Return nil to indicate error
	}

	for _, param := range fn.Params {
		paramType, err := semantic.DeriveSemanticType(param.Type, cm)
		if err != nil {
			colors.RED.Printf("Error deriving type for parameter '%s': %s\n", param.Identifier.Name, err.Error())
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, param.Type.Loc(), "Invalid parameter type: "+err.Error(), report.RESOLVER_PHASE)
			return nil // Return nil to indicate error
		}
		paramTypes = append(paramTypes, paramType)

		// Try to update parameter symbol first (for function declarations)
		// If it doesn't exist, create it (for function literals)
		createOrUpdateParameterSymbol(&param, paramType, functionScope, r)
	}
	return paramTypes
}

func createOrUpdateParameterSymbol(param *ast.Parameter, paramType stype.Type, functionScope *symbol.SymbolTable, r *analyzer.AnalyzerNode) {
	if functionScope != nil && param.Identifier != nil {
		// Try to find existing parameter symbol (from collector phase for function declarations)
		if paramSymbol, found := functionScope.Lookup(param.Identifier.Name); found {
			// Update existing symbol
			paramSymbol.Type = paramType
			if r.Debug {
				colors.YELLOW.Printf("Updated parameter symbol '%s' with type '%s'\n", param.Identifier.Name, paramType.String())
			}
		} else {
			// Create new symbol (for function literals)
			paramSymbol := symbol.NewSymbolWithLocation(param.Identifier.Name, symbol.SymbolVar, paramType, param.Identifier.Loc())
			err := functionScope.Declare(param.Identifier.Name, paramSymbol)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, param.Identifier.Loc(), "Failed to declare parameter symbol: "+err.Error(), report.RESOLVER_PHASE)
			} else if r.Debug {
				colors.GREEN.Printf("Created parameter symbol '%s' with type '%s' for function literal\n", param.Identifier.Name, paramType.String())
			}
		}
	}
}

func resolveReturnType(r *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, cm *modules.Module) stype.Type {
	if fn != nil && fn.ReturnType != nil {
		retType, err := semantic.DeriveSemanticType(fn.ReturnType, cm)
		if err != nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.ReturnType.Loc(), "Invalid return type: "+err.Error(), report.RESOLVER_PHASE)
			return nil
		}
		return retType
	}
	return &stype.PrimitiveType{Name: types.VOID}
}

func resolveVariableDeclaration(r *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *modules.Module) {

	colors.ORANGE.Printf("Resolving variable declaration\n")

	for i, variable := range decl.Variables {

		colors.BLUE.Printf("Resolving variable declaration '%s' at %s\n", variable.Identifier.Name, variable.Identifier.Loc().String())

		// Check initializer expression if present
		if i < len(decl.Initializers) && decl.Initializers[i] != nil {
			resolveExpr(r, decl.Initializers[i], cm)
		}

		// Look up the already-declared symbol from the collector phase
		symbol, found := cm.SymbolTable.Lookup(variable.Identifier.Name)
		if !found {
			colors.RED.Printf("Variable '%s' not found in symbol table\n", variable.Identifier.Name)
			r.Ctx.Reports.AddCriticalError(r.Program.FullPath, variable.Identifier.Loc(), "Variable '"+variable.Identifier.Name+"' was not collected during symbol collection phase", report.RESOLVER_PHASE)
			continue
		}

		if variable.ExplicitType != nil {
			got, err := semantic.DeriveSemanticType(variable.ExplicitType, cm)
			if err != nil {
				colors.RED.Printf("Error deriving type for variable '%s': %s\n", variable.Identifier.Name, err.Error())
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.ExplicitType.Loc(), "Invalid explicit type for variable declaration: "+err.Error(), report.RESOLVER_PHASE)
				continue
			}
			// Update the symbol's type
			symbol.Type = got
			if r.Debug {
				colors.TEAL.Printf("Declared variable symbol '%s' with explicit type '%v' at %s\n", variable.Identifier.Name, symbol.Type, variable.Identifier.Loc().String())
			}
		}
	}
}

func resolveTypeDeclaration(r *analyzer.AnalyzerNode, decl *ast.TypeDeclStmt, cm *modules.Module) {
	aliasName := decl.Alias.Name
	if aliasName == "" {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.Alias.Loc(), "Type alias name cannot be empty", report.RESOLVER_PHASE)
		return
	}

	// Look up the already-declared symbol from the collector phase
	symbol, found := cm.SymbolTable.Lookup(aliasName)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.Alias.Loc(), "Type alias '"+aliasName+"' was not collected during symbol collection phase", report.RESOLVER_PHASE)
		return
	}

	typeToDeclare, err := semantic.DeriveSemanticType(decl.BaseType, cm)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.BaseType.Loc(), "Invalid base type for type declaration: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	symbolType := &stype.UserType{
		Name:       types.TYPE_NAME(aliasName),
		Definition: typeToDeclare,
	}

	// Update the symbol's type
	symbol.Type = symbolType

	if r.Debug {
		colors.ORANGE.Printf("Resolved type alias '%v', Def: %v at %s\n", symbol.Type, symbol.Type.(*stype.UserType).Definition, decl.Alias.Loc().String())
	}
}

func resolveAssignmentStmt(r *analyzer.AnalyzerNode, assign *ast.AssignmentStmt, cm *modules.Module) {
	// Resolve left-hand side expressions (assignees)
	if assign.Left != nil {
		resolveExpressionList(r, assign.Left, cm)
	}
	// Resolve right-hand side expressions (values)
	if assign.Right != nil {
		resolveExpressionList(r, assign.Right, cm)
	}

	if r.Debug {
		colors.TEAL.Printf("Resolved assignment statement at %s\n", assign.Loc().String())
	}
}

// resolveMethodDecl resolves method declarations stored in struct scopes
func resolveMethodDecl(r *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module) {
	// Get the receiver type name to find the struct's scope
	receiverTypeName := ""
	if method.Receiver.Type != nil {
		receiverTypeName = string(method.Receiver.Type.Type())
	}
	methodName := method.Method.Name

	// Find the struct type symbol to get its scope
	structSymbol, found := cm.SymbolTable.Lookup(receiverTypeName)
	if !found {
		colors.RED.Printf("Struct type '%s' not found in symbol table\n", receiverTypeName)
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Struct type '"+receiverTypeName+"' is not declared", report.RESOLVER_PHASE)
		return
	}

	if structSymbol.Scope == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Struct type '"+receiverTypeName+"' does not have a scope for methods", report.RESOLVER_PHASE)
		return
	}

	// Look for the method in the struct's scope
	methodSymbol, found := structSymbol.Scope.Lookup(methodName)
	if !found {
		colors.RED.Printf("Method '%s' not found in struct '%s' scope\n", methodName, receiverTypeName)
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Method '"+methodName+"' is not declared for struct '"+receiverTypeName+"'", report.RESOLVER_PHASE)
		return
	}

	// Get method scope from the method symbol itself
	methodScope := methodSymbol.Scope
	if methodScope == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Method scope for '"+methodName+"' not found", report.RESOLVER_PHASE)
		return
	}

	// Resolve receiver type
	receiverType, err := semantic.DeriveSemanticType(method.Receiver.Type, cm)
	if err != nil {
		colors.RED.Printf("Error deriving type for receiver '%s': %s\n", method.Receiver.Identifier.Name, err.Error())
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Receiver.Type.Loc(), "Invalid receiver type: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	// Update receiver symbol with resolved type
	if receiverSymbol, found := methodScope.Lookup(method.Receiver.Identifier.Name); found {
		receiverSymbol.Type = receiverType
		if r.Debug {
			colors.GREEN.Printf("Updated receiver symbol '%s' with type '%v\n", method.Receiver.Identifier.Name, receiverType)
		}
	}

	// Resolve parameter types and update method scope symbols
	paramTypes := resolveParameterTypes(r, method.Function, cm, methodScope)
	if paramTypes == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Failed to resolve parameter types for method '"+methodName+"'", report.RESOLVER_PHASE)
		return // Error occurred during parameter resolution
	}

	// Resolve return type
	returnType := resolveReturnType(r, method.Function, cm)
	if returnType == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Failed to resolve return type for method '"+methodName+"'", report.RESOLVER_PHASE)
		return // Error occurred during return type resolution
	}

	fmt.Printf("Resolved method '%s.%s' with receiver: %v, parameters: %v and return type: %v\n", receiverTypeName, methodName, receiverType, paramTypes, returnType)

	// Resolve method body
	//set scope
	originalTable := cm.SymbolTable
	cm.SymbolTable = methodScope
	resolveBlock(r, method.Function.Body, cm)
	// Restore original module scope
	cm.SymbolTable = originalTable

	// Create and assign method type (same as function type but includes receiver information)
	methodType := stype.FunctionType{
		Parameters: paramTypes,
		ReturnType: returnType,
	}
	methodSymbol.Type = &methodType
}

func resolveFunctionLiteral(r *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, cm *modules.Module) {
	if fn == nil || fn.ID == "" {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function literal missing ID", report.RESOLVER_PHASE)
		return
	}

	// Create a function symbol for this literal
	functionSymbol := symbol.NewSymbolWithLocation(fn.ID, symbol.SymbolFunc, nil, fn.Loc())

	// Create function scope with module scope as parent
	functionScope := symbol.NewSymbolTable(cm.SymbolTable)
	functionSymbol.Scope = functionScope

	// Add function literal symbol to module symbol table
	if err := cm.SymbolTable.Declare(fn.ID, functionSymbol); err != nil {
		// If it already exists, get the existing symbol
		existingSymbol, found := cm.SymbolTable.Lookup(fn.ID)
		if !found {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to declare function literal symbol: "+err.Error(), report.RESOLVER_PHASE)
			return
		}
		functionSymbol = existingSymbol
		functionScope = functionSymbol.Scope
		if functionScope == nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function literal scope not found", report.RESOLVER_PHASE)
			return
		}
	}

	if r.Debug {
		colors.BLUE.Printf("Resolved function literal '%s' at %s\n", fn.ID, fn.Loc().String())
	}

	// Resolve parameter types and update function scope symbols
	paramTypes := resolveParameterTypes(r, fn, cm, functionScope)
	if paramTypes == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to resolve parameter types for function literal '"+fn.ID+"'", report.RESOLVER_PHASE)
		return
	}

	// Resolve return type
	returnType := resolveReturnType(r, fn, cm)
	if returnType == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to resolve return type for function literal '"+fn.ID+"'", report.RESOLVER_PHASE)
		return
	}

	originalTable := cm.SymbolTable
	cm.SymbolTable = functionScope
	// Resolve function body
	resolveBlock(r, fn.Body, cm)
	// Restore original module scope
	cm.SymbolTable = originalTable

	// Create and assign function type
	functionType := stype.FunctionType{
		Parameters: paramTypes,
		ReturnType: returnType,
	}
	functionSymbol.Type = &functionType

	if r.Debug {
		colors.ORANGE.Printf("Resolved function literal '%s' with parameters: %v and return type: %v\n", fn.ID, paramTypes, returnType)
	}
}
