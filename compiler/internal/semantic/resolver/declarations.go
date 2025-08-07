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

func resolveVariableDeclaration(r *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *modules.Module) {

	colors.ORANGE.Printf("Resolving variable declaration\n")

	for i, variable := range decl.Variables {

		colors.BLUE.Printf("Resolving variable declaration '%s' at %s\n", variable.Identifier.Name, variable.Identifier.Loc())

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

// resolveMethodDecl resolves method declarations stored in struct scopes
func resolveMethodDecl(r *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module) {
	if !validateMethodReceiver(r, method, cm) {
		return
	}

	receiverTypeName := getReceiverTypeName(method)
	methodName := method.Method.Name

	structSymbol, methodSymbol, methodScope := getMethodSymbols(r, method, cm, receiverTypeName, methodName)
	if structSymbol == nil || methodSymbol == nil || methodScope == nil {
		return
	}

	receiverType := resolveMethodReceiver(r, method, cm, methodScope)
	if receiverType == nil {
		return
	}

	paramTypes, returnType := resolveMethodSignature(r, method, cm, methodScope, methodName)
	if paramTypes == nil || returnType == nil {
		return
	}

	fmt.Printf("Resolved method '%s.%s' with receiver: %v, parameters: %v and return type: %v\n", receiverTypeName, methodName, receiverType, paramTypes, returnType)

	resolveMethodBody(r, method, cm, methodScope)
	assignMethodType(methodSymbol, paramTypes, returnType)
}

// validateMethodReceiver validates that the method receiver is valid for method definitions
func validateMethodReceiver(r *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module) bool {
	if method.Receiver == nil || method.Receiver.Type == nil {
		return true
	}

	if !isValidMethodReceiverType(method.Receiver.Type) {
		// Skip processing - error already reported in collector phase for non-UserDefinedType
		return false
	}

	// For UserDefinedType, we need to check if it resolves to a struct
	if userDefinedType, ok := method.Receiver.Type.(*ast.UserDefinedType); ok {
		if !isUserDefinedTypeValidForMethods(r, userDefinedType, cm) {
			// Skip processing - error reported in the validation function
			return false
		}
	}

	return true
}

// getReceiverTypeName extracts the receiver type name from the method declaration
func getReceiverTypeName(method *ast.MethodDecl) string {
	if method.Receiver != nil && method.Receiver.Type != nil {
		return string(method.Receiver.Type.Type())
	}
	return ""
}

// getMethodSymbols retrieves the struct symbol, method symbol, and method scope
func getMethodSymbols(r *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module, receiverTypeName, methodName string) (*symbol.Symbol, *symbol.Symbol, *symbol.SymbolTable) {
	// Find the struct type symbol to get its scope
	structSymbol, found := cm.SymbolTable.Lookup(receiverTypeName)
	if !found {
		colors.RED.Printf("Struct type '%s' not found in symbol table\n", receiverTypeName)
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Struct type '"+receiverTypeName+"' is not declared", report.RESOLVER_PHASE)
		return nil, nil, nil
	}

	if structSymbol.Scope == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Struct type '"+receiverTypeName+"' does not have a scope for methods", report.RESOLVER_PHASE)
		return nil, nil, nil
	}

	// Look for the method in the struct's scope
	methodSymbol, found := structSymbol.Scope.Lookup(methodName)
	if !found {
		colors.RED.Printf("Method '%s' not found in struct '%s' scope\n", methodName, receiverTypeName)
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Method '"+methodName+"' is not declared for struct '"+receiverTypeName+"'", report.RESOLVER_PHASE)
		return nil, nil, nil
	}

	// Get method scope from the method symbol itself
	methodScope := methodSymbol.Scope
	if methodScope == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Method scope for '"+methodName+"' not found", report.RESOLVER_PHASE)
		return nil, nil, nil
	}

	return structSymbol, methodSymbol, methodScope
}

// resolveMethodReceiver resolves the receiver type and updates the receiver symbol
func resolveMethodReceiver(r *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module, methodScope *symbol.SymbolTable) stype.Type {
	// Resolve receiver type
	receiverType, err := semantic.DeriveSemanticType(method.Receiver.Type, cm)
	if err != nil {
		colors.RED.Printf("Error deriving type for receiver '%s': %s\n", method.Receiver.Identifier.Name, err.Error())
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Receiver.Type.Loc(), "Invalid receiver type: "+err.Error(), report.RESOLVER_PHASE)
		return nil
	}

	// Update receiver symbol with resolved type
	if receiverSymbol, found := methodScope.Lookup(method.Receiver.Identifier.Name); found {
		receiverSymbol.Type = receiverType
		if r.Debug {
			colors.GREEN.Printf("Updated receiver symbol '%s' with type '%v\n", method.Receiver.Identifier.Name, receiverType)
		}
	}

	return receiverType
}

// resolveMethodSignature resolves parameter types and return type for the method
func resolveMethodSignature(r *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module, methodScope *symbol.SymbolTable, methodName string) ([]stype.Type, stype.Type) {
	// Resolve parameter types and update method scope symbols
	paramTypes := resolveParameterTypes(r, method.Function, cm, methodScope)
	if paramTypes == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Failed to resolve parameter types for method '"+methodName+"'", report.RESOLVER_PHASE)
		return nil, nil
	}

	// Resolve return type
	returnType := resolveReturnType(r, method.Function, cm)
	if returnType == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, method.Loc(), "Failed to resolve return type for method '"+methodName+"'", report.RESOLVER_PHASE)
		return nil, nil
	}

	return paramTypes, returnType
}

// resolveMethodBody resolves the method body within the method scope
func resolveMethodBody(r *analyzer.AnalyzerNode, method *ast.MethodDecl, cm *modules.Module, methodScope *symbol.SymbolTable) {
	originalTable := cm.SymbolTable
	cm.SymbolTable = methodScope
	resolveBlock(r, method.Function.Body, cm)
	cm.SymbolTable = originalTable
}

// assignMethodType creates and assigns the method type to the method symbol
func assignMethodType(methodSymbol *symbol.Symbol, paramTypes []stype.Type, returnType stype.Type) {
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
	case *ast.StructType:
		// Anonymous struct types are not allowed for methods
		return false
	case *ast.IntType, *ast.FloatType, *ast.StringType, *ast.ByteType, *ast.BoolType:
		// Primitive types are not allowed for methods
		return false
	case *ast.ArrayType:
		// Array types are not allowed for methods
		return false
	case *ast.InterfaceType:
		// Interface types are not allowed for method definitions
		return false
	case *ast.FunctionType:
		// Function types are not allowed for methods
		return false
	default:
		// Unknown type, not allowed
		return false
	}
}

// isUserDefinedTypeValidForMethods checks if a UserDefinedType resolves to a struct type
// and can have methods defined on it
func isUserDefinedTypeValidForMethods(r *analyzer.AnalyzerNode, userType *ast.UserDefinedType, cm *modules.Module) bool {
	// Resolve the user-defined type to its semantic type
	resolvedType, err := semantic.DeriveSemanticType(userType, cm)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			userType.Loc(),
			"Cannot resolve type '"+string(userType.TypeName)+"': "+err.Error(),
			report.RESOLVER_PHASE,
		)
		return false
	}

	// Check if the resolved type ultimately resolves to a struct
	if !isResolvedTypeAStruct(resolvedType) {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			userType.Loc(),
			"Cannot define methods on type '"+string(userType.TypeName)+"' because it does not resolve to a struct type",
			report.RESOLVER_PHASE,
		).AddHint("Methods can only be defined on struct types or type aliases that resolve to struct types")
		return false
	}

	return true
}

// isResolvedTypeAStruct checks if a resolved semantic type is ultimately a struct type
// This handles unwrapping of type aliases to find the underlying type
func isResolvedTypeAStruct(t stype.Type) bool {
	// Use the unwrapping utility from semantic package
	unwrapped := semantic.UnwrapType(t)
	_, isStruct := unwrapped.(*stype.StructType)
	return isStruct
}
