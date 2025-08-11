package typecheck

import (
	"ferret/internal/frontend/ast"
	"ferret/internal/modules"
	"ferret/internal/semantic"
	"ferret/internal/semantic/analyzer"
	"ferret/internal/semantic/stype"
	"ferret/internal/types"
	"ferret/internal/utils/msg"
	"ferret/report"
	"fmt"
)

// checkIdentifierType gets the type of an identifier from the symbol table
func checkIdentifierType(e *ast.IdentifierExpr, cm *modules.Module) stype.Type {
	if sym, found := cm.SymbolTable.Lookup(e.Name); found {
		return sym.Type
	}
	return nil
}

// checkBinaryExprType infers the result type of binary expressions
func checkBinaryExprType(r *analyzer.AnalyzerNode, e *ast.BinaryExpr, cm *modules.Module) stype.Type {
	leftType := evaluateExpressionType(r, *e.Left, cm)
	rightType := evaluateExpressionType(r, *e.Right, cm)

	if leftType == nil || rightType == nil {
		return nil
	}

	leftType = semantic.UnwrapType(leftType)   // Unwrap any type aliases
	rightType = semantic.UnwrapType(rightType) // Unwrap any type aliases

	resultType := getBinaryOperationResultType(e.Operator.Value, leftType, rightType)
	if resultType == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("invalid binary operation: %s %s %s", leftType, e.Operator.Value, rightType),
			report.TYPECHECK_PHASE,
		)
	}

	return resultType
}

// getBinaryOperationResultType determines the result type of binary operations
func getBinaryOperationResultType(operator string, left, right stype.Type) stype.Type {
	switch operator {
	case "**", "+", "-", "*", "/", "%":
		return getArithmeticResultType(operator, left, right)
	case "==", "!=", "<", "<=", ">", ">=":
		return getComparisonResultType(left, right)
	case "&&", "||":
		return getLogicalResultType(left, right)
	case "&", "|", "^", "<<", ">>":
		return getBitwiseResultType(left, right)
	default:
		return nil
	}
}

// getArithmeticResultType handles arithmetic operations
func getArithmeticResultType(operator string, left, right stype.Type) stype.Type {
	// String concatenation
	if operator == "+" && semantic.IsStringType(left) && semantic.IsStringType(right) {
		return &stype.PrimitiveType{TypeName: types.STRING}
	}

	// Numeric operations
	if semantic.IsNumericType(left) && semantic.IsNumericType(right) {
		return getCommonNumericType(left, right)
	}

	return nil
}

// getComparisonResultType handles comparison operations
func getComparisonResultType(left, right stype.Type) stype.Type {

	leftToRight, _ := isImplicitCastable(left, right)
	rightToLeft, _ := isImplicitCastable(right, left)

	if leftToRight || rightToLeft {
		return &stype.PrimitiveType{TypeName: types.BOOL}
	}

	return nil
}

// getLogicalResultType handles logical operations
func getLogicalResultType(left, right stype.Type) stype.Type {
	if semantic.IsBoolType(left) && semantic.IsBoolType(right) {
		return &stype.PrimitiveType{TypeName: types.BOOL}
	}
	return nil
}

// getBitwiseResultType handles bitwise operations
func getBitwiseResultType(left, right stype.Type) stype.Type {
	if semantic.IsIntegerType(left) && semantic.IsIntegerType(right) {
		return getCommonNumericType(left, right)
	}
	return nil
}

// getCommonNumericType finds the common type for numeric operations
func getCommonNumericType(left, right stype.Type) stype.Type {

	leftPrim, leftOk := left.(*stype.PrimitiveType)
	rightPrim, rightOk := right.(*stype.PrimitiveType)

	if !leftOk || !rightOk {
		return nil
	}

	// Promotion hierarchy: higher numbers win
	hierarchy := map[types.TYPE_NAME]int{
		types.INT8: 1, types.UINT8: 1, types.BYTE: 1,
		types.INT16: 2, types.UINT16: 2,
		types.INT32: 3, types.UINT32: 3,
		types.INT64: 4, types.UINT64: 4,
		types.FLOAT32: 5, types.FLOAT64: 6,
	}

	leftLevel, leftExists := hierarchy[leftPrim.TypeName]
	rightLevel, rightExists := hierarchy[rightPrim.TypeName]

	if !leftExists || !rightExists {
		return nil
	}

	if leftLevel >= rightLevel {
		return left
	}
	return right
}

// checkArrayLiteralType infers array literal types
func checkArrayLiteralType(r *analyzer.AnalyzerNode, e *ast.ArrayLiteralExpr, cm *modules.Module) stype.Type {
	if len(e.Elements) == 0 {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			e.Loc(),
			"cannot infer array type from empty array literal",
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Get type from first element
	elementType := evaluateExpressionType(r, e.Elements[0], cm)
	if elementType == nil {
		return nil
	}

	// Verify all elements are compatible
	for _, element := range e.Elements[1:] {
		elemType := evaluateExpressionType(r, element, cm)
		if elemType == nil {
			continue
		}

		if ok, err := isImplicitCastable(elementType, elemType); !ok {

			semanticError := r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				element.Loc(),
				fmt.Sprintf("error: %s\narray elements must be of type %s, but got %s", err.Error(), elementType, elemType),
				report.TYPECHECK_PHASE,
			)

			if ok, _ := isExplicitCastable(elemType, elementType); !ok {
				semanticError.AddHint(msg.CastHint(elementType))
			}

			return nil
		}
	}

	return &stype.ArrayType{ElementType: elementType}
}

// checkIndexableType infers types for array/map indexing
func checkIndexableType(r *analyzer.AnalyzerNode, e *ast.IndexableExpr, cm *modules.Module) stype.Type {
	indexableType := evaluateExpressionType(r, *e.Indexable, cm)
	if indexableType == nil {
		return nil
	}

	indexType := evaluateExpressionType(r, *e.Index, cm)
	if indexType == nil {
		return nil
	}

	if !semantic.IsIntegerType(indexType) {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			(*e.Index).Loc(),
			"array index must be an integer type",
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Check if it's an array
	if arrayType, ok := indexableType.(*stype.ArrayType); ok {
		return arrayType.ElementType
	}

	// allow strings
	if stringType, ok := indexableType.(*stype.PrimitiveType); ok && stringType.TypeName == types.STRING {
		return &stype.PrimitiveType{TypeName: types.BYTE}
	}

	r.Ctx.Reports.AddSemanticError(
		r.Program.FullPath,
		(*e.Indexable).Loc(),
		fmt.Sprintf("type '%s' is not indexable", indexableType),
		report.TYPECHECK_PHASE,
	)
	return nil
}

func checkUnaryExprType(r *analyzer.AnalyzerNode, e *ast.UnaryExpr, cm *modules.Module) stype.Type {
	operandType := evaluateExpressionType(r, *e.Operand, cm)
	if operandType == nil {
		return nil
	}

	// Handle specific unary operations
	switch e.Operator.Value {
	case "!":
		if semantic.IsBoolType(operandType) {
			return operandType // Boolean negation returns same type
		}
	case "-":
		if semantic.IsNumericType(operandType) {
			return operandType // Unary minus returns same numeric type
		}
	default:
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("unsupported unary operator '%s' for type '%s'", e.Operator.Value, operandType),
			report.TYPECHECK_PHASE,
		)
	}

	return nil
}

// checkPrefixExprType handles prefix increment/decrement operations (++x, --x)
func checkPrefixExprType(r *analyzer.AnalyzerNode, e *ast.PrefixExpr, cm *modules.Module) stype.Type {
	operandType := evaluateExpressionType(r, *e.Operand, cm)
	if operandType == nil {
		return nil
	}

	// Handle prefix operations
	switch e.Operator.Value {
	case "++", "--":
		if semantic.IsNumericType(operandType) {
			return operandType // Prefix increment/decrement returns same numeric type
		}
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("operator '%s' cannot be applied to type '%s'", e.Operator.Value, operandType),
			report.TYPECHECK_PHASE,
		)
	default:
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("unsupported prefix operator '%s'", e.Operator.Value),
			report.TYPECHECK_PHASE,
		)
	}

	return nil
}

// checkPostfixExprType handles postfix increment/decrement operations (x++, x--)
func checkPostfixExprType(r *analyzer.AnalyzerNode, e *ast.PostfixExpr, cm *modules.Module) stype.Type {
	operandType := evaluateExpressionType(r, *e.Operand, cm)
	if operandType == nil {
		return nil
	}

	// Handle postfix operations
	switch e.Operator.Value {
	case "++", "--":
		if semantic.IsNumericType(operandType) {
			return operandType // Postfix increment/decrement returns same numeric type
		}
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("operator '%s' cannot be applied to type '%s'", e.Operator.Value, operandType),
			report.TYPECHECK_PHASE,
		)
	default:
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("unsupported postfix operator '%s'", e.Operator.Value),
			report.TYPECHECK_PHASE,
		)
	}

	return nil
}
