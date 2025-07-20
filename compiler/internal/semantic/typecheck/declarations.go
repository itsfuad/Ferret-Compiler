package typecheck

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"fmt"
)

func checkVariableDeclaration(r *analyzer.AnalyzerNode, varDecl *ast.VarDeclStmt, cm *ctx.Module) {
	for i, variable := range varDecl.Variables {

		//if initializer not provided, skip type inference
		if len(varDecl.Initializers) == 0 {
			return
		}

		variableInModule, _ := cm.SymbolTable.Lookup(variable.Identifier.Name)
		initializer := varDecl.Initializers[i]
		typeToAdd := evaluateExpressionType(r, initializer, cm)

		if variable.ExplicitType == nil {
			variableInModule.Type = typeToAdd
			if r.Debug {
				colors.CYAN.Printf("Inferred type for variable '%s': %s\n", variable.Identifier.Name, typeToAdd)
			}
		} else if variable.ExplicitType != nil && typeToAdd != nil {
			//both explicit type and initializer are provided. they must match
			explicitType, err := ctx.DeriveSemanticType(variable.ExplicitType, cm)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.ExplicitType.Loc(), "Invalid explicit type for variable declaration: "+err.Error(), report.TYPECHECK_PHASE)
				return
			}
			if !IsAssignableFrom(explicitType, typeToAdd) {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, initializer.Loc(), fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'", typeToAdd.String(), variable.Identifier.Name, explicitType.String()), report.TYPECHECK_PHASE)
				return
			}
		}
	}
}

func checkAssignmentStmt(r *analyzer.AnalyzerNode, assign *ast.AssignmentStmt, cm *ctx.Module) {
	// Check that we have both left and right hand sides
	if assign.Left == nil || assign.Right == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(), "Assignment statement must have both left and right hand sides", report.TYPECHECK_PHASE)
		return
	}

	leftExprs := *assign.Left
	rightExprs := *assign.Right

	// Check that the number of assignees matches the number of values
	if len(leftExprs) != len(rightExprs) {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(),
			fmt.Sprintf("assignment mismatch: %d variables but %d values", len(leftExprs), len(rightExprs)),
			report.TYPECHECK_PHASE)
		return
	}

	// Type check each assignment pair
	for i := 0; i < len(leftExprs); i++ {
		lhs := leftExprs[i]
		rhs := rightExprs[i]

		// Get the type of the right-hand side expression
		rhsType := evaluateExpressionType(r, rhs, cm)
		if rhsType == nil {
			continue // Error already reported in evaluateExpressionType
		}

		// For left-hand side, we need to check if it's a valid assignee
		switch lhsExpr := lhs.(type) {
		case *ast.IdentifierExpr:
			// Check if the identifier exists and get its type
			symbol, found := cm.SymbolTable.Lookup(lhsExpr.Name)
			if !found {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, lhsExpr.Loc(),
					fmt.Sprintf("undefined variable: %s", lhsExpr.Name),
					report.TYPECHECK_PHASE)
				continue
			}

			// Check if the variable type is compatible with the assigned value
			if symbol.Type != nil && !IsAssignableFrom(symbol.Type, rhsType) {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(),
					fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'",
						rhsType.String(), lhsExpr.Name, symbol.Type.String()),
					report.TYPECHECK_PHASE)
				continue
			}

			if r.Debug {
				colors.CYAN.Printf("Assignment type check: variable '%s' (%s) = %s\n",
					lhsExpr.Name, symbol.Type, rhsType)
			}

		case *ast.IndexableExpr:
			// Handle array/slice element assignment
			lhsType := evaluateExpressionType(r, lhs, cm)
			if lhsType != nil && !IsAssignableFrom(lhsType, rhsType) {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(),
					fmt.Sprintf("cannot assign value of type '%s' to indexable element of type '%s'",
						rhsType.String(), lhsType.String()),
					report.TYPECHECK_PHASE)
				continue
			}

		case *ast.FieldAccessExpr:
			// Handle struct field assignment
			lhsType := evaluateExpressionType(r, lhs, cm)
			if lhsType != nil && !IsAssignableFrom(lhsType, rhsType) {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, assign.Loc(),
					fmt.Sprintf("cannot assign value of type '%s' to field of type '%s'",
						rhsType.String(), lhsType.String()),
					report.TYPECHECK_PHASE)
				continue
			}

		default:
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, lhs.Loc(),
				fmt.Sprintf("invalid left-hand side in assignment: %T", lhsExpr),
				report.TYPECHECK_PHASE)
		}
	}

	if r.Debug {
		colors.TEAL.Printf("Type checked assignment statement at %s\n", assign.Loc().String())
	}
}
