package typecheck

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/semantic"
	"compiler/internal/frontend/semantic/analyzer"
	"compiler/internal/frontend/semantic/stype"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/types"
)

// ===== CORE ASSIGNABILITY CHECK =====

// IsAssignableFrom checks if a value of type 'source' can be assigned to 'target'
// Note: This function has limited type resolution capability. For full alias resolution,
// the type checker should use resolveTypeAlias with analyzer context.
func IsAssignableFrom(target, source stype.Type) bool {
	// Exact type match
	if target.Equals(source) {
		return true
	}

	// Handle user types (aliases) - limited resolution without symbol table access
	resolvedTarget := semantic.UnwrapType(target)
	resolvedSource := semantic.UnwrapType(source)

	colors.PURPLE.Printf("Checking assignability: %v → %v ", resolvedSource, resolvedTarget)

	if resolvedTarget.Equals(resolvedSource) || isNumericPromotion(resolvedTarget, resolvedSource) || isArrayCompatible(resolvedTarget, resolvedSource) || isFunctionCompatible(resolvedTarget, resolvedSource) || isStructCompatible(resolvedTarget, resolvedSource) {
		colors.GREEN.Println(" ✔ ")
		return true
	}

	colors.RED.Println(" ✘ ")

	return false
}

// ===== HELPER FUNCTIONS =====

// isNumericPromotion checks if source can be promoted to target (implicit conversion)
func isNumericPromotion(target, source stype.Type) bool {
	targetPrim, targetOk := target.(*stype.PrimitiveType)
	sourcePrim, sourceOk := source.(*stype.PrimitiveType)

	if !targetOk || !sourceOk {
		return false
	}

	// Define promotion rules
	promotions := map[types.TYPE_NAME][]types.TYPE_NAME{
		// Integer promotions (smaller -> larger)
		types.INT16:  {types.INT8},
		types.INT32:  {types.INT8, types.INT16},
		types.INT64:  {types.INT8, types.INT16, types.INT32},
		types.UINT16: {types.UINT8, types.BYTE},
		types.UINT32: {types.UINT8, types.UINT16, types.BYTE},
		types.UINT64: {types.UINT8, types.UINT16, types.UINT32, types.BYTE},

		// Float promotions (int -> float, smaller float -> larger float)
		types.FLOAT32: {types.INT8, types.INT16, types.UINT8, types.UINT16, types.BYTE},
		types.FLOAT64: {types.INT8, types.INT16, types.INT32, types.UINT8, types.UINT16, types.UINT32, types.BYTE, types.FLOAT32},
	}

	if allowedSources, exists := promotions[targetPrim.Name]; exists {
		for _, allowedSource := range allowedSources {
			if sourcePrim.Name == allowedSource {
				return true
			}
		}
	}

	return false
}

// isArrayCompatible checks array type compatibility
func isArrayCompatible(target, source stype.Type) bool {
	targetArray, targetOk := target.(*stype.ArrayType)
	sourceArray, sourceOk := source.(*stype.ArrayType)

	if !targetOk || !sourceOk {
		return false
	}

	return IsAssignableFrom(targetArray.ElementType, sourceArray.ElementType)
}

// isFunctionCompatible checks function type compatibility
func isFunctionCompatible(target, source stype.Type) bool {
	targetFunc, targetOk := target.(*stype.FunctionType)
	sourceFunc, sourceOk := source.(*stype.FunctionType)

	if !targetOk || !sourceOk {
		return false
	}

	// Parameter and return type counts must match
	if len(targetFunc.Parameters) != len(sourceFunc.Parameters) {
		return false
	}

	// Parameters are contravariant, returns are covariant
	for i := range targetFunc.Parameters {
		if !IsAssignableFrom(sourceFunc.Parameters[i], targetFunc.Parameters[i]) {
			return false
		}
	}

	// Compare return types
	if targetFunc.ReturnType == nil && sourceFunc.ReturnType == nil {
		return true
	}
	if targetFunc.ReturnType == nil || sourceFunc.ReturnType == nil {
		return false
	}

	return IsAssignableFrom(targetFunc.ReturnType, sourceFunc.ReturnType)
}

// isStructCompatible checks structural compatibility of structs
func isStructCompatible(target, source stype.Type) bool {
	targetStruct, targetOk := target.(*stype.StructType)
	sourceStruct, sourceOk := source.(*stype.StructType)

	if !targetOk || !sourceOk {
		return false
	}

	// Source must have all fields that target requires, with compatible types
	for fieldName, targetFieldType := range targetStruct.Fields {
		sourceFieldType, exists := sourceStruct.Fields[fieldName]
		if !exists || !IsAssignableFrom(targetFieldType, sourceFieldType) {
			return false
		}
	}

	return true
}

// ===== EXPRESSION TYPE INFERENCE HELPERS =====

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
			"invalid binary operation: "+leftType.String()+" "+e.Operator.Value+" "+rightType.String(),
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
		return &stype.PrimitiveType{Name: types.STRING}
	}

	// Numeric operations
	if semantic.IsNumericType(left) && semantic.IsNumericType(right) {
		return getCommonNumericType(left, right)
	}

	return nil
}

// getComparisonResultType handles comparison operations
func getComparisonResultType(left, right stype.Type) stype.Type {
	if IsAssignableFrom(left, right) || IsAssignableFrom(right, left) {
		return &stype.PrimitiveType{Name: types.BOOL}
	}
	return nil
}

// getLogicalResultType handles logical operations
func getLogicalResultType(left, right stype.Type) stype.Type {
	if semantic.IsBoolType(left) && semantic.IsBoolType(right) {
		return &stype.PrimitiveType{Name: types.BOOL}
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
	if left.Equals(right) {
		return left
	}

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

	leftLevel, leftExists := hierarchy[leftPrim.Name]
	rightLevel, rightExists := hierarchy[rightPrim.Name]

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

		if !IsAssignableFrom(elementType, elemType) {
			// Try to find common type
			commonType := getCommonNumericType(elementType, elemType)
			if commonType == nil {
				r.Ctx.Reports.AddSemanticError(
					r.Program.FullPath,
					element.Loc(),
					"array element type mismatch",
					report.TYPECHECK_PHASE,
				)
				return nil
			}
			elementType = commonType
		}
	}

	return &stype.ArrayType{ElementType: elementType, Name: types.ARRAY}
}

// checkIndexableType infers types for array/map indexing
func checkIndexableType(r *analyzer.AnalyzerNode, e *ast.IndexableExpr, cm *modules.Module) stype.Type {
	indexableType := evaluateExpressionType(r, *e.Indexable, cm)
	if indexableType == nil {
		return nil
	}

	// Check if it's an array
	if arrayType, ok := indexableType.(*stype.ArrayType); ok {
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

		return arrayType.ElementType
	}

	r.Ctx.Reports.AddSemanticError(
		r.Program.FullPath,
		(*e.Indexable).Loc(),
		"cannot index non-array type",
		report.TYPECHECK_PHASE,
	)
	return nil
}
