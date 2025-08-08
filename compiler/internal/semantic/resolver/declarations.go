package resolver

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic"
	"ferret/compiler/internal/semantic/analyzer"
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/symbol"
	"ferret/compiler/internal/types"
	"fmt"
)

func resolveFunctionDecl(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) {

	functionSymbol, found := cm.SymbolTable.Lookup(fn.Identifier.Name)
	if !found {
		colors.RED.Printf("Function '%s' not found in symbol table\n", fn.Identifier.Name)
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function '"+fn.Identifier.Name+"' is not declared", report.RESOLVER_PHASE)
		return
	}

	colors.PINK.Printf("Resolving function declaration '%s' on %s scope at %s\n", fn.Identifier.Name, functionSymbol.SelfScope.ScopeName, fn.Loc())

	resolveFunctionLike(r, fn.Function, functionSymbol, cm)	
}

func resolveFunctionLike(r *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, functionSymbol *symbol.Symbol, cm *modules.Module) {
	// Resolve parameter types and update function scope symbols
	paramTypes := resolveParameterTypes(r, fn, cm, functionSymbol.SelfScope)
	if paramTypes == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to resolve parameter types for function '"+fn.ID+"'", report.RESOLVER_PHASE)
		return // Error occurred during parameter resolution
	}

	// Resolve return type
	returnType := resolveReturnType(r, fn, cm)
	if returnType == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to resolve return type for function '"+fn.ID+"'", report.RESOLVER_PHASE)
		return // Error occurred during return type resolution
	}

	fmt.Printf("Resolved function '%s' with parameters: %v and return type: %v\n", fn.ID, paramTypes, returnType)
	// Resolve function body
	originalTable := cm.SymbolTable
	cm.SymbolTable = functionSymbol.SelfScope // Temporarily switch to function scope
	resolveBlock(r, fn.Body, cm)
	cm.SymbolTable = originalTable // Restore original module scope
	// Create and assign function type
	functionType := &stype.FunctionType{
		Parameters: paramTypes,
		ReturnType: returnType,
	}
	functionSymbol.Type = functionType
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
				colors.YELLOW.Printf("Updated parameter symbol '%s' with type '%s'\n", param.Identifier.Name, paramType)
			}
		} else {
			// Create new symbol (for function literals)
			paramSymbol := symbol.NewSymbolWithLocation(param.Identifier.Name, symbol.SymbolVar, paramType, param.Identifier.Loc())
			err := functionScope.Declare(param.Identifier.Name, paramSymbol)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, param.Identifier.Loc(), "Failed to declare parameter symbol: "+err.Error(), report.RESOLVER_PHASE)
			} else if r.Debug {
				colors.GREEN.Printf("Created parameter symbol '%s' with type '%s' for function literal\n", param.Identifier.Name, paramType)
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
	return &stype.PrimitiveType{TypeName: types.VOID} // Default return type if none specified
}

func resolveMethodDecl(r *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module) {
	colors.ORANGE.Printf("Resolving method declaration '%s' at %s\n", method.Method.Name, method.Loc())

	//get the receiver symbol
	receiverSymbol, found := cm.SymbolTable.Lookup(method.Receiver.Type.Type().String())
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Receiver.Identifier.Loc(), fmt.Sprintf("Receiver '%s' not found in method '%s'", method.Receiver.Identifier.Name, method.Method.Name), report.RESOLVER_PHASE)
		return
	}

	// resolve the method
	methodSymbol, found := receiverSymbol.SelfScope.Lookup(method.Method.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Method.Loc(), fmt.Sprintf("Method '%s' not collected during symbol collection phase", method.Method.Name), report.RESOLVER_PHASE)
		return
	}

	// Check if the receiver type is valid for methods
	if !isValidMethodReceiverType(method.Receiver.Type) {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Receiver.Type.Loc(), "Invalid receiver type for method declaration", report.RESOLVER_PHASE)
		return
	}

	receiverType, err := semantic.DeriveSemanticType(method.Receiver.Type, cm)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Receiver.Type.Loc(), "Invalid receiver type: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	colors.PINK.Printf("Resolving method '%s' on receiver type '%s' at %s\n", method.Method.Name, receiverSymbol.Type, method.Method.Loc())

	receiverSymbol.Type = receiverType // Update receiver type
	receiverParam, found := methodSymbol.SelfScope.Lookup(method.Receiver.Identifier.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Receiver.Identifier.Loc(), fmt.Sprintf("Receiver parameter '%s' was not collected during symbol collection phase", method.Receiver.Identifier.Name), report.RESOLVER_PHASE)
		return
	}

	// Update the receiver parameter type
	receiverParam.Type = receiverType

	resolveFunctionLike(r, method.Function, methodSymbol, cm)
}

func resolveVariableDeclaration(r *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *modules.Module) {

	colors.ORANGE.Printf("Resolving variable declaration from %s\n", cm.SymbolTable.ScopeName)

	for i, variable := range decl.Variables {

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
				colors.TEAL.Printf("Declared variable symbol '%s' with explicit type '%v' at %s\n", variable.Identifier.Name, symbol.Type, variable.Identifier.Loc())
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
		Name:       aliasName,
		Definition: typeToDeclare,
	}

	// Update the symbol's type
	symbol.Type = symbolType

	if r.Debug {
		colors.ORANGE.Printf("Resolved type alias '%v', Def: %v at %s\n", symbol.Type, symbol.Type.(*stype.UserType).Definition, decl.Alias.Loc())
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
		colors.TEAL.Printf("Resolved assignment statement at %s\n", assign.Loc())
	}
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

	// Add function literal symbol to module symbol table
	if err := cm.SymbolTable.Declare(fn.ID, functionSymbol); err != nil {
		existingSymbol, found := cm.SymbolTable.Lookup(fn.ID)
		if !found {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to declare function literal symbol: "+err.Error(), report.RESOLVER_PHASE)
			return
		}
		// If it already exists, get the existing symbol (So compiler wont crash and we can still resolve it)
		functionSymbol = existingSymbol
	}

	if r.Debug {
		colors.BLUE.Printf("Resolved function literal '%s' at %s\n", fn.ID, fn.Loc())
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

// isValidMethodReceiverType checks if a DataType is valid for method receiver
// Only named struct types (UserDefinedType) are allowed
func isValidMethodReceiverType(dataType ast.DataType) bool {
	switch dataType.(type) {
	case *ast.UserDefinedType:
		// This is a named type, which is valid (it should resolve to a struct)
		return true
	default:
		// Unknown type, not allowed
		return false
	}
}