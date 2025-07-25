package typecheck

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/stype"
	"fmt"
)

func checkVariableDeclaration(r *analyzer.AnalyzerNode, varDecl *ast.VarDeclStmt, cm *ctx.Module) {
	// If no initializers, skip inference
	if len(varDecl.Initializers) == 0 {
		return
	}

	for i, variable := range varDecl.Variables {
		checkSingleVariableDeclaration(r, variable, varDecl.Initializers[i], cm)
	}
}

func checkSingleVariableDeclaration(r *analyzer.AnalyzerNode, variable *ast.VariableToDeclare, initializer ast.Expression, cm *ctx.Module) {
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
		return
	}

	// Case: both explicit type and initializer → validate compatibility
	checkExplicitTypeCompatibility(r, variable, inferredType, initializer, cm)
}

func checkExplicitTypeCompatibility(r *analyzer.AnalyzerNode, variable *ast.VariableToDeclare, inferredType stype.Type, initializer ast.Expression, cm *ctx.Module) {
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

	if !IsAssignableFrom(explicitType, inferredType) {
		rp := r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			initializer.Loc(),
			fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'",
				inferredType.String(), variable.Identifier.Name, explicitType.String()),
			report.TYPECHECK_PHASE,
		)

		if isCastValid(inferredType, explicitType) {
			rp.AddHint(fmt.Sprintf("Want to cast😐 ? Write `as %s` after the expression", explicitType.String()))
		}
	}
}

func checkAssignmentStmt(r *analyzer.AnalyzerNode, assign *ast.AssignmentStmt, cm *ctx.Module) {
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
		if !IsAssignableFrom(lhsType, rhsType) {
			rp := r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Right.Loc(), fmt.Sprintf("cannot assign value of type '%s' to assignee of type '%s'", rhsType.String(), lhsType.String()), report.TYPECHECK_PHASE)

			if isCastValid(rhsType, lhsType) {
				rp.AddHint(fmt.Sprintf("Want to cast😐 ? Write `as %s` after the expression", lhsType.String()))
			}

			continue
		}
	}

	if r.Debug {
		colors.TEAL.Printf("Type checked assignment statement at %s\n", assign.Loc().String())
	}
}

func checkExprListType(r *analyzer.AnalyzerNode, exprs *ast.ExpressionList, cm *ctx.Module) []stype.Type {
	return checkExprListTypeWithContext(r, exprs, cm, false) // Don't allow void by default (for assignments)
}

func checkExprListTypeWithContext(r *analyzer.AnalyzerNode, exprs *ast.ExpressionList, cm *ctx.Module, allowVoid bool) []stype.Type {
	var types []stype.Type
	for _, expr := range *exprs {
		exprType := evaluateExpressionType(r, expr, cm)
		if exprType == nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, expr.Loc(), "undefined expression type", report.TYPECHECK_PHASE)
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
