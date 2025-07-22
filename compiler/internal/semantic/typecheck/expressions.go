package typecheck

import (
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/stype"
	atype "compiler/internal/types"
	"fmt"
)

// evaluateExpressionType infers the semantic type from an AST expression
func evaluateExpressionType(r *analyzer.AnalyzerNode, expr ast.Expression, cm *ctx.Module) stype.Type {
	if expr == nil {
		return nil
	}

	var resultType stype.Type

	switch e := expr.(type) {
	// Literals
	case *ast.StringLiteral:
		resultType = &stype.PrimitiveType{Name: atype.STRING}
	case *ast.IntLiteral:
		resultType = &stype.PrimitiveType{Name: atype.INT32}
	case *ast.FloatLiteral:
		resultType = &stype.PrimitiveType{Name: atype.FLOAT64}
	case *ast.BoolLiteral:
		resultType = &stype.PrimitiveType{Name: atype.BOOL}
	case *ast.ByteLiteral:
		resultType = &stype.PrimitiveType{Name: atype.BYTE}

	// Complex expressions
	case *ast.IdentifierExpr:
		resultType = checkIdentifierType(e, cm)
	case *ast.BinaryExpr:
		resultType = checkBinaryExprType(r, e, cm)
	case *ast.ArrayLiteralExpr:
		resultType = checkArrayLiteralType(r, e, cm)
	case *ast.IndexableExpr:
		resultType = checkIndexableType(r, e, cm)
	case *ast.VarScopeResolution:
		resultType = checkImportedSymbolType(r, e, cm)
	case *ast.FunctionCallExpr:
		resultType = checkFunctionCallType(r, e, cm)
	case *ast.CastExpr:
		resultType = checkCastExprType(r, e, cm)

	default:
		// Unknown expression type
		resultType = nil
		r.Ctx.Reports.AddCriticalError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("Unsupported expression type <%T> for type inference", e),
			report.TYPECHECK_PHASE,
		)
	}

	return resultType
}

func checkFunctionCallType(r *analyzer.AnalyzerNode, call *ast.FunctionCallExpr, cm *ctx.Module) stype.Type {
	// Get the type of the function being called
	functionType := evaluateExpressionType(r, *call.Caller, cm)
	if functionType == nil {
		return nil
	}

	// Verify it's a function type
	funcType, ok := functionType.(*stype.FunctionType)
	if !ok {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			call.Loc(),
			"cannot call non-function type: "+functionType.String(),
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Check argument count
	expectedCount := len(funcType.Parameters)
	actualCount := len(call.Arguments)

	if expectedCount != actualCount {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			call.Loc(),
			fmt.Sprintf("function expects %d arguments, but %d were provided", expectedCount, actualCount),
			report.TYPECHECK_PHASE,
		)
		return funcType.ReturnType // Return the expected return type even with wrong arg count
	}

	// Check argument types
	for i, arg := range call.Arguments {
		argType := evaluateExpressionType(r, arg, cm)
		if argType == nil {
			continue // Skip if we can't determine argument type
		}

		expectedType := funcType.Parameters[i]
		if !IsAssignableFrom(expectedType, argType) {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				call.Loc(),
				fmt.Sprintf("argument %d: cannot use %s as %s", i+1, argType.String(), expectedType.String()),
				report.TYPECHECK_PHASE,
			)
		}
	}

	// Return the function's return type (single return type now)
	return funcType.ReturnType
}

// checkCastExprType validates type cast expressions and returns the target type
func checkCastExprType(r *analyzer.AnalyzerNode, cast *ast.CastExpr, cm *ctx.Module) stype.Type {
	// Evaluate the source expression type
	sourceType := evaluateExpressionType(r, *cast.Value, cm)
	if sourceType == nil {
		return nil
	}

	// Convert AST target type to semantic type
	targetType, err := semantic.DeriveSemanticType(cast.TargetType, cm)
	if err != nil || targetType == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			cast.Loc(),
			fmt.Sprintf("invalid target type in cast expression: %v", err),
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Check if the cast is valid
	if !isCastValid(sourceType, targetType) {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			cast.Loc(),
			fmt.Sprintf("cannot cast from '%s' to '%s'", sourceType.String(), targetType.String()),
			report.TYPECHECK_PHASE,
		)
		return targetType // Still return target type for further analysis
	}

	return targetType
}

// isCastValid determines if a cast from sourceType to targetType is valid
func isCastValid(sourceType, targetType stype.Type) bool {
	// Allow casting between same types (no-op cast)
	if sourceType.String() == targetType.String() {
		return true
	}

	sourcePrim, sourceOk := sourceType.(*stype.PrimitiveType)
	targetPrim, targetOk := targetType.(*stype.PrimitiveType)

	// Both types must be primitive types for casting
	if !sourceOk || !targetOk {
		return false
	}

	// Allow ALL numeric to numeric casting with explicit "as" keyword
	// The developer explicitly requests the conversion, so allow both widening and narrowing
	if atype.IsNumericTypeName(sourcePrim.Name) && atype.IsNumericTypeName(targetPrim.Name) {
		return true
	}

	// Special case: byte can be cast to/from u8 and i8
	if sourcePrim.Name == atype.BYTE {
		return targetPrim.Name == atype.UINT8 || targetPrim.Name == atype.INT8
	}
	if targetPrim.Name == atype.BYTE {
		return sourcePrim.Name == atype.UINT8 || sourcePrim.Name == atype.INT8
	}

	// No valid cast found
	return false
}
