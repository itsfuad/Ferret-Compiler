package typecheck

import (
	"ferret/colors"
	"ferret/internal/frontend/ast"
	"ferret/internal/modules"
	"ferret/internal/semantic"
	"ferret/internal/semantic/analyzer"
	"ferret/internal/semantic/stype"
	"ferret/internal/symbol"
	"ferret/internal/utils/msg"
	"ferret/report"
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
			colors.TEAL.Printf("Inferred type '%s' for variable '%s' at %s\n", inferredType, variable.Identifier.Name, variable.Identifier.Loc())
		}
		return
	}

	// Case: both explicit type and initializer → validate compatibility
	checkTypeCompatibility(r, variable, inferredType, initializer, cm)
}

func checkTypeCompatibility(r *analyzer.AnalyzerNode, variable *ast.VariableToDeclare, inferredType stype.Type, initializer ast.Expression, cm *modules.Module) {
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

	if ok, err := isImplicitCastable(explicitType, inferredType); !ok {
		rp := r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			initializer.Loc(),
			fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s': %s",
				inferredType, variable.Identifier.Name, explicitType, err.Error()),
			report.TYPECHECK_PHASE,
		)

		if ok, _ := isExplicitCastable(inferredType, explicitType); ok {
			rp.AddHint(msg.CastHint(explicitType))
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
		if ok, err := isImplicitCastable(lhsType, rhsType); !ok {
			rp := r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Right.Loc(), fmt.Sprintf("cannot assign value of type '%s' to assignee of type '%s': %s", rhsType, lhsType, err.Error()), report.TYPECHECK_PHASE)
			if ok, _ := isExplicitCastable(rhsType, lhsType); ok {
				rp.AddHint(msg.CastHint(lhsType))
			}
			continue
		}
	}

	if r.Debug {
		colors.TEAL.Printf("Type checked assignment statement at %s\n", assign.Loc())
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

	// Check the function literal using the helper function
	checkFunctionLiteral(r, funcDecl.Function, cm, functionSymbol.SelfScope)
}

func checkMethodDecl(r *analyzer.AnalyzerNode, methodDecl *ast.MethodDecl, cm *modules.Module) {

	//get receiver symbol
	receiverSymbol, found := cm.SymbolTable.Lookup(methodDecl.Receiver.Type.Type().String())
	if !found {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			methodDecl.Receiver.Type.Loc(),
			fmt.Sprintf("Receiver type '%s' not found in symbol table", methodDecl.Receiver.Type.Type().String()),
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Get the method symbol and its scope
	methodSymbol, found := receiverSymbol.SelfScope.Lookup(methodDecl.Method.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			methodDecl.Loc(),
			fmt.Sprintf("Method '%s' not found in symbol table", methodDecl.Method.Name),
			report.TYPECHECK_PHASE,
		)
		return
	}

	// Check the method literal using the helper function
	checkFunctionLiteral(r, methodDecl.Function, cm, methodSymbol.SelfScope)
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

	// Check the function literal using the helper function
	checkFunctionLiteral(r, fn, cm, functionSymbol.SelfScope)

	// Return the function's type
	return functionSymbol.Type
}
