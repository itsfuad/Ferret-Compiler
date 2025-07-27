package typecheck

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/stype"
	atype "compiler/internal/types"
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

	colors.MAGENTA.Printf("Checking assignability: %v → %v\n", resolvedSource, resolvedTarget)

	if resolvedTarget.Equals(resolvedSource) {
		return true
	}

	// Check numeric promotions
	if isNumericPromotion(resolvedTarget, resolvedSource) {
		return true
	}

	// Check array compatibility
	if isArrayCompatible(resolvedTarget, resolvedSource) {
		return true
	}

	// Check function compatibility
	if isFunctionCompatible(resolvedTarget, resolvedSource) {
		return true
	}

	// Check struct compatibility (structural typing)
	if isStructCompatible(resolvedTarget, resolvedSource) {
		return true
	}

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
	promotions := map[atype.TYPE_NAME][]atype.TYPE_NAME{
		// Integer promotions (smaller -> larger)
		atype.INT16:  {atype.INT8},
		atype.INT32:  {atype.INT8, atype.INT16},
		atype.INT64:  {atype.INT8, atype.INT16, atype.INT32},
		atype.UINT16: {atype.UINT8, atype.BYTE},
		atype.UINT32: {atype.UINT8, atype.UINT16, atype.BYTE},
		atype.UINT64: {atype.UINT8, atype.UINT16, atype.UINT32, atype.BYTE},

		// Float promotions (int -> float, smaller float -> larger float)
		atype.FLOAT32: {atype.INT8, atype.INT16, atype.UINT8, atype.UINT16, atype.BYTE},
		atype.FLOAT64: {atype.INT8, atype.INT16, atype.INT32, atype.UINT8, atype.UINT16, atype.UINT32, atype.BYTE, atype.FLOAT32},
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
func checkIdentifierType(e *ast.IdentifierExpr, cm *ctx.Module) stype.Type {
	if sym, found := cm.SymbolTable.Lookup(e.Name); found {
		return sym.Type
	}
	return nil
}

// checkBinaryExprType infers the result type of binary expressions
func checkBinaryExprType(r *analyzer.AnalyzerNode, e *ast.BinaryExpr, cm *ctx.Module) stype.Type {
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
		return &stype.PrimitiveType{Name: atype.STRING}
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
		return &stype.PrimitiveType{Name: atype.BOOL}
	}
	return nil
}

// getLogicalResultType handles logical operations
func getLogicalResultType(left, right stype.Type) stype.Type {
	if semantic.IsBoolType(left) && semantic.IsBoolType(right) {
		return &stype.PrimitiveType{Name: atype.BOOL}
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
	hierarchy := map[atype.TYPE_NAME]int{
		atype.INT8: 1, atype.UINT8: 1, atype.BYTE: 1,
		atype.INT16: 2, atype.UINT16: 2,
		atype.INT32: 3, atype.UINT32: 3,
		atype.INT64: 4, atype.UINT64: 4,
		atype.FLOAT32: 5, atype.FLOAT64: 6,
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
func checkArrayLiteralType(r *analyzer.AnalyzerNode, e *ast.ArrayLiteralExpr, cm *ctx.Module) stype.Type {
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

	return &stype.ArrayType{ElementType: elementType, Name: atype.ARRAY}
}

// checkIndexableType infers types for array/map indexing
func checkIndexableType(r *analyzer.AnalyzerNode, e *ast.IndexableExpr, cm *ctx.Module) stype.Type {
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
