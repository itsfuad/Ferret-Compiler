package typecheck

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic"
	"ferret/compiler/internal/semantic/analyzer"
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/symbol"
	"fmt"
)

func checkVariableDeclaration(r *analyzer.AnalyzerNode, varDecl *ast.VarDeclStmt, cm *modules.Module) {
	// If no initializers, skip inference
	if len(varDecl.Initializers) == 0 {
		return
	}

	for i, variable := range varDecl.Variables {
		checkSingleVariableDeclaration(r, variable, varDecl.Initializers[i], cm)
	}
}

func checkSingleVariableDeclaration(r *analyzer.AnalyzerNode, variable *ast.VariableToDeclare, initializer ast.Expression, cm *modules.Module) {
	variableInModule, _ := cm.SymbolTable.Lookup(variable.Identifier.Name)
	inferredType := evaluateExpressionType(r, initializer, cm)

	// Check if we're trying to assign void to a variable
	if inferredType != nil && semantic.IsVoidType(inferredType) {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			initializer.Loc(),
			fmt.Sprintf("cannot assign void expression to variable '%s'", variable.Identifier.Name),
			report.TYPECHECK_PHASE,
		).AddHint("Void means no type. Not even null. So, a variable cannot be void. It must have a valid type.")
		return
	}

	// Case: no explicit type → just infer
	if variable.ExplicitType == nil {
		variableInModule.Type = inferredType
		if r.Debug {
			colors.TEAL.Printf("Inferred type '%s' for variable '%s' at %s\n", inferredType, variable.Identifier.Name, variable.Identifier.Loc().String())
		}
		return
	}

	// Case: both explicit type and initializer → validate compatibility
	checkExplicitTypeCompatibility(r, variable, inferredType, initializer, cm)
}

func checkExplicitTypeCompatibility(r *analyzer.AnalyzerNode, variable *ast.VariableToDeclare, inferredType stype.Type, initializer ast.Expression, cm *modules.Module) {
	explicitType, err := semantic.DeriveSemanticType(variable.ExplicitType, cm)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			variable.ExplicitType.Loc(),
			"Invalid explicit type for variable declaration: "+err.Error(),
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Check if someone is trying to explicitly declare a variable as void
	if semantic.IsVoidType(explicitType) {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			variable.ExplicitType.Loc(),
			fmt.Sprintf("variable '%s' cannot be declared with void type", variable.Identifier.Name),
			report.TYPECHECK_PHASE,
		)
		return
	}

	if !isImplicitCastable(explicitType, inferredType) {
		rp := r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			initializer.Loc(),
			fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'",
				inferredType.String(), variable.Identifier.Name, explicitType),
			report.TYPECHECK_PHASE,
		)

		if ok, _ := isExplicitCastable(inferredType, explicitType); ok {
			rp.AddHint(fmt.Sprintf("Want to cast😐 ? Write `as %s` after the expression", explicitType))
		}
	}
}

func checkAssignmentStmt(r *analyzer.AnalyzerNode, assign *ast.AssignmentStmt, cm *modules.Module) {
	// Check that we have both left and right hand sides
	if assign.Left == nil || assign.Right == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(), "Assignment statement must have both left and right hand sides", report.TYPECHECK_PHASE)
		return
	}

	leftTypes := checkExprListType(r, assign.Left, cm)
	rightTypes := checkExprListType(r, assign.Right, cm)

	if len(leftTypes) == 0 {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Left.Loc(), "No valid left-hand side expressions found", report.TYPECHECK_PHASE)
		return
	}

	if len(rightTypes) == 0 {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Right.Loc(), "No valid right-hand side expressions found", report.TYPECHECK_PHASE)
		return
	}

	// Check that the number of assignees matches the number of values
	if len(leftTypes) != len(rightTypes) {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(),
			fmt.Sprintf("assignment mismatch: %d variable(s) but %d value(s)", len(leftTypes), len(rightTypes)),
			report.TYPECHECK_PHASE)
		return
	}

	for i, lhs := range *assign.Left {
		rhsType := rightTypes[i]
		lhsType := leftTypes[i]
		if lhsType == nil || rhsType == nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, lhs.Loc(), "Failed to determine type for assignment", report.TYPECHECK_PHASE)
			continue
		}
		if !isImplicitCastable(lhsType, rhsType) {
			rp := r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Right.Loc(), fmt.Sprintf("cannot assign value of type '%s' to assignee of type '%s'", rhsType.String(), lhsType.String()), report.TYPECHECK_PHASE)

			if ok, _ := isExplicitCastable(rhsType, lhsType); ok {
				rp.AddHint(fmt.Sprintf("Want to cast😐 ? Write `as %s` after the expression", lhsType.String()))
			}

			continue
		}
	}

	if r.Debug {
		colors.TEAL.Printf("Type checked assignment statement at %s\n", assign.Loc().String())
	}
}

func checkExprListType(r *analyzer.AnalyzerNode, exprs *ast.ExpressionList, cm *modules.Module) []stype.Type {
	return checkExprListTypeWithContext(r, exprs, cm, false) // Don't allow void by default (for assignments)
}

func checkExprListTypeWithContext(r *analyzer.AnalyzerNode, exprs *ast.ExpressionList, cm *modules.Module, allowVoid bool) []stype.Type {
	var types []stype.Type
	for _, expr := range *exprs {
		exprType := evaluateExpressionType(r, expr, cm)
		if exprType == nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, expr.Loc(), "invalid expression", report.TYPECHECK_PHASE)
			continue
		}

		// Check if void is not allowed (assignment context) and we have a void type
		if !allowVoid && semantic.IsVoidType(exprType) {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, expr.Loc(), "void expressions cannot be used in assignment context", report.TYPECHECK_PHASE)
			continue
		}

		types = append(types, exprType)
	}
	return types
}

// checkFunctionDecl validates function declarations and their return paths
func checkFunctionDecl(r *analyzer.AnalyzerNode, funcDecl *ast.FunctionDecl, cm *modules.Module) {
	if funcDecl.Function == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			funcDecl.Loc(),
			"Function declaration missing function body",
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Get function symbol and its scope
	functionSymbol, found := cm.SymbolTable.Lookup(funcDecl.Identifier.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			funcDecl.Loc(),
			fmt.Sprintf("Function '%s' not found in symbol table", funcDecl.Identifier.Name),
			report.TYPECHECK_PHASE,
		)
		return
	}

	if functionSymbol.Scope == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			funcDecl.Loc(),
			fmt.Sprintf("Function scope for '%s' not found", funcDecl.Identifier.Name),
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Check the function literal using the helper function
	checkFunctionLiteral(r, funcDecl.Function, cm, functionSymbol.Scope)
}

// checkFunctionLiteral validates function literals and their return paths
func checkFunctionLiteral(r *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, cm *modules.Module, functionScope *symbol.SymbolTable) {
	// Get the expected return type
	var expectedReturnType stype.Type = nil
	if fn.ReturnType != nil {
		resolvedType, err := semantic.DeriveSemanticType(fn.ReturnType, cm)
		if err != nil {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				fn.ReturnType.Loc(),
				fmt.Sprintf("Invalid return type: %s", err.Error()),
				report.TYPECHECK_PHASE,
			)
			return
		}
		expectedReturnType = resolvedType
	}

	// Switch to function scope for type checking if we have one
	// Temporarily switch to function scope
	originalTable := cm.SymbolTable
	// Ensure function scope has access to module imports
	functionScope.Imports = originalTable.Imports
	cm.SymbolTable = functionScope

	// Analyze the function body for control flow and returns
	result := analyzeControlFlow(r, fn.Body, cm, expectedReturnType)

	// Restore module scope
	cm.SymbolTable = originalTable

	// Check if non-void function has all paths returning
	if expectedReturnType != nil && !isVoidType(expectedReturnType) && !result.AllPathsReturn {
		if len(result.CriticalMissingReturns) > 0 {
			// Report main error
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				fn.Loc(),
				"Not all paths in function return a value",
				report.TYPECHECK_PHASE,
			)

			// Add specific errors for each critical missing return location
			for i, loc := range result.CriticalMissingReturns {
				if i < 3 { // Limit to first 3 locations to avoid spam
					r.Ctx.Reports.AddSemanticError(
						r.Program.FullPath,
						&loc,
						"Missing return statement in this path",
						report.TYPECHECK_PHASE,
					)
				}
			}
		} else {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				fn.Loc(),
				"Not all paths in function return a value",
				report.TYPECHECK_PHASE,
			)
		}
	}
}

func checkMethodDecl(r *analyzer.AnalyzerNode, methodDecl *ast.MethodDecl, cm *modules.Module) {
	if methodDecl.Function == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			methodDecl.Loc(),
			"Method declaration missing function body",
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Validate that methods can only be defined on struct types
	if methodDecl.Receiver != nil && methodDecl.Receiver.Type != nil {
		if !isValidMethodReceiverType(methodDecl.Receiver.Type) {
			// Skip processing - error already reported in collector phase
			return
		}
	}

	// Get the receiver type name to find the struct's scope
	receiverTypeName := ""
	if methodDecl.Receiver.Type != nil {
		receiverTypeName = string(methodDecl.Receiver.Type.Type())
	}
	methodName := methodDecl.Method.Name

	// Find the struct type symbol to get its scope
	structSymbol, found := cm.SymbolTable.Lookup(receiverTypeName)
	if !found {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			methodDecl.Loc(),
			"Struct type '"+receiverTypeName+"' not found in symbol table",
			report.TYPECHECK_PHASE,
		)
		return
	}

	if structSymbol.Scope == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			methodDecl.Loc(),
			"Struct type '"+receiverTypeName+"' does not have a scope for methods",
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Look for the method in the struct's scope
	methodSymbol, found := structSymbol.Scope.Lookup(methodName)
	if !found {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			methodDecl.Loc(),
			"Method '"+methodName+"' not found in struct '"+receiverTypeName+"' scope",
			report.TYPECHECK_PHASE,
		)
		return
	}

	if methodSymbol.Scope == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			methodDecl.Loc(),
			"Method scope for '"+methodName+"' not found",
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Check the method function using the helper function
	checkFunctionLiteral(r, methodDecl.Function, cm, methodSymbol.Scope)
}

// checkFunctionLiteralType checks function literals and returns their type
func checkFunctionLiteralType(r *analyzer.AnalyzerNode, fn *ast.FunctionLiteral, cm *modules.Module) stype.Type {
	if fn == nil || fn.ID == "" {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function literal missing ID", report.TYPECHECK_PHASE)
		return nil
	}

	// Get function symbol and its scope
	functionSymbol, found := cm.SymbolTable.Lookup(fn.ID)
	if !found {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			fn.Loc(),
			"Function literal '"+fn.ID+"' not found in symbol table",
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Get function scope from the function symbol itself
	functionScope := functionSymbol.Scope
	if functionScope == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			fn.Loc(),
			"Function literal scope for '"+fn.ID+"' not found",
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Check the function literal using the helper function
	checkFunctionLiteral(r, fn, cm, functionScope)

	// Return the function's type
	return functionSymbol.Type
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
