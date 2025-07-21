package typecheck

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/types"
	"fmt"
)

func checkVariableDeclaration(r *analyzer.AnalyzerNode, varDecl *ast.VarDeclStmt, cm *ctx.Module) {
	// If no initializers, skip inference
	if len(varDecl.Initializers) == 0 {
		return
	}

	for i, variable := range varDecl.Variables {
		variableInModule, _ := cm.SymbolTable.Lookup(variable.Identifier.Name)
		initializer := varDecl.Initializers[i]
		inferredType := evaluateExpressionType(r, initializer, cm)

		// Case: no explicit type → just infer
		if variable.ExplicitType == nil {
			variableInModule.Type = inferredType
			if r.Debug {
				colors.CYAN.Printf("Inferred type for variable '%s': %s\n", variable.Identifier.Name, inferredType)
			}
			continue
		}

		// Case: both explicit type and initializer → validate compatibility
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

		if !IsAssignableFrom(explicitType, inferredType) {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				initializer.Loc(),
				fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'",
					inferredType.String(), variable.Identifier.Name, explicitType.String()),
				report.TYPECHECK_PHASE,
			)
			return
		}
	}
}

func checkAssignmentStmt(r *analyzer.AnalyzerNode, assign *ast.AssignmentStmt, cm *ctx.Module) {
	// Check that we have both left and right hand sides
	if assign.Left == nil || assign.Right == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(), "Assignment statement must have both left and right hand sides", report.TYPECHECK_PHASE)
		return
	}

	// Check that the number of assignees matches the number of values
	if len(*assign.Left) != len(*assign.Right) {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(),
			fmt.Sprintf("assignment mismatch: %d variables but %d values", len(*assign.Left), len(*assign.Right)),
			report.TYPECHECK_PHASE)
		return
	}

	leftTypes := checkExprListType(r, assign.Left, cm)
	rightTypes := checkExprListType(r, assign.Right, cm)

	for i, lhs := range *assign.Left {
		rhsType := rightTypes[i]
		lhsType := leftTypes[i]
		if lhsType == nil || rhsType == nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, lhs.Loc(), "Failed to determine type for assignment", report.TYPECHECK_PHASE)
			continue
		}
		if !IsAssignableFrom(lhsType, rhsType) {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, lhs.Loc(), fmt.Sprintf("cannot assign value of type '%s' to assignee of type '%s'", rhsType.String(), lhsType.String()), report.TYPECHECK_PHASE)
			continue
		}
	}

	if r.Debug {
		colors.TEAL.Printf("Type checked assignment statement at %s\n", assign.Loc().String())
	}
}

func checkExprListType(r *analyzer.AnalyzerNode, exprs *ast.ExpressionList, cm *ctx.Module) []types.Type {
	var types []types.Type
	for _, expr := range *exprs {
		exprType := evaluateExpressionType(r, expr, cm)
		if exprType == nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, expr.Loc(), "Failed to determine expression type", report.TYPECHECK_PHASE)
			continue
		}
		types = append(types, exprType)
	}
	return types
}
