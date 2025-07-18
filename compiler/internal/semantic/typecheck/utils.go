package typecheck

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/types"
)

// getDatatype converts an AST DataType to a semantic Type
func getDatatype(astType ast.DataType) semantic.Type {
	return semantic.ASTToSemanticType(astType)
}

// inferTypeFromExpression infers the semantic type from an AST expression
func inferTypeFromExpression(r *analyzer.AnalyzerNode, expr ast.Expression, cm *ctx.Module) semantic.Type {
	if expr == nil {
		return nil
	}

	var resultType semantic.Type

	switch e := expr.(type) {
	// Literals
	case *ast.StringLiteral:
		resultType = &semantic.PrimitiveType{Name: types.STRING}
	case *ast.IntLiteral:
		resultType = &semantic.PrimitiveType{Name: types.INT32}
	case *ast.FloatLiteral:
		resultType = &semantic.PrimitiveType{Name: types.FLOAT64}
	case *ast.BoolLiteral:
		resultType = &semantic.PrimitiveType{Name: types.BOOL}
	case *ast.ByteLiteral:
		resultType = &semantic.PrimitiveType{Name: types.BYTE}

	// Complex expressions
	case *ast.IdentifierExpr:
		resultType = inferIdentifierType(e, cm)
	case *ast.BinaryExpr:
		resultType = inferBinaryExprType(r, e, cm)
	case *ast.ArrayLiteralExpr:
		resultType = inferArrayLiteralType(r, e, cm)
	case *ast.IndexableExpr:
		resultType = inferIndexableType(r, e, cm)

	default:
		// Unknown expression type
		resultType = nil
	}

	// Debug logging
	if r.Debug && resultType != nil {
		colors.YELLOW.Printf("Inferred type for expression: %s\n", resultType.String())
	}

	return resultType
}

// ===== CORE ASSIGNABILITY CHECK =====

// IsAssignableFrom checks if a value of type 'source' can be assigned to 'target'
func IsAssignableFrom(target, source semantic.Type) bool {
	// Exact type match
	if target.Equals(source) {
		return true
	}

	// Handle user types (aliases)
	resolvedTarget := resolveUserType(target)
	resolvedSource := resolveUserType(source)

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

// resolveUserType resolves user types to their underlying types
func resolveUserType(t semantic.Type) semantic.Type {
	if userType, ok := t.(*semantic.UserType); ok && userType.Definition != nil {
		return resolveUserType(userType.Definition)
	}
	return t
}

// isNumericPromotion checks if source can be promoted to target (implicit conversion)
func isNumericPromotion(target, source semantic.Type) bool {
	targetPrim, targetOk := target.(*semantic.PrimitiveType)
	sourcePrim, sourceOk := source.(*semantic.PrimitiveType)

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
func isArrayCompatible(target, source semantic.Type) bool {
	targetArray, targetOk := target.(*semantic.ArrayType)
	sourceArray, sourceOk := source.(*semantic.ArrayType)

	if !targetOk || !sourceOk {
		return false
	}

	return IsAssignableFrom(targetArray.ElementType, sourceArray.ElementType)
}

// isFunctionCompatible checks function type compatibility
func isFunctionCompatible(target, source semantic.Type) bool {
	targetFunc, targetOk := target.(*semantic.FunctionType)
	sourceFunc, sourceOk := source.(*semantic.FunctionType)

	if !targetOk || !sourceOk {
		return false
	}

	// Parameter and return type counts must match
	if len(targetFunc.Parameters) != len(sourceFunc.Parameters) ||
		len(targetFunc.ReturnTypes) != len(sourceFunc.ReturnTypes) {
		return false
	}

	// Parameters are contravariant, returns are covariant
	for i := range targetFunc.Parameters {
		if !IsAssignableFrom(sourceFunc.Parameters[i], targetFunc.Parameters[i]) {
			return false
		}
	}

	for i := range targetFunc.ReturnTypes {
		if !IsAssignableFrom(targetFunc.ReturnTypes[i], sourceFunc.ReturnTypes[i]) {
			return false
		}
	}

	return true
}

// isStructCompatible checks structural compatibility of structs
func isStructCompatible(target, source semantic.Type) bool {
	targetStruct, targetOk := target.(*semantic.StructType)
	sourceStruct, sourceOk := source.(*semantic.StructType)

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

// inferIdentifierType gets the type of an identifier from the symbol table
func inferIdentifierType(e *ast.IdentifierExpr, cm *ctx.Module) semantic.Type {
	if sym, found := cm.SymbolTable.Lookup(e.Name); found {
		return sym.Type
	}
	return nil
}

// inferBinaryExprType infers the result type of binary expressions
func inferBinaryExprType(r *analyzer.AnalyzerNode, e *ast.BinaryExpr, cm *ctx.Module) semantic.Type {
	leftType := inferTypeFromExpression(r, *e.Left, cm)
	rightType := inferTypeFromExpression(r, *e.Right, cm)

	if leftType == nil || rightType == nil {
		return nil
	}

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
func getBinaryOperationResultType(operator string, left, right semantic.Type) semantic.Type {
	switch operator {
	case "+", "-", "*", "/", "%":
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
func getArithmeticResultType(operator string, left, right semantic.Type) semantic.Type {
	// String concatenation
	if operator == "+" && semantic.IsStringType(left) && semantic.IsStringType(right) {
		return &semantic.PrimitiveType{Name: types.STRING}
	}

	// Numeric operations
	if semantic.IsNumericType(left) && semantic.IsNumericType(right) {
		return getCommonNumericType(left, right)
	}

	return nil
}

// getComparisonResultType handles comparison operations
func getComparisonResultType(left, right semantic.Type) semantic.Type {
	if IsAssignableFrom(left, right) || IsAssignableFrom(right, left) {
		return &semantic.PrimitiveType{Name: types.BOOL}
	}
	return nil
}

// getLogicalResultType handles logical operations
func getLogicalResultType(left, right semantic.Type) semantic.Type {
	if semantic.IsBoolType(left) && semantic.IsBoolType(right) {
		return &semantic.PrimitiveType{Name: types.BOOL}
	}
	return nil
}

// getBitwiseResultType handles bitwise operations
func getBitwiseResultType(left, right semantic.Type) semantic.Type {
	if semantic.IsIntegerType(left) && semantic.IsIntegerType(right) {
		return getCommonNumericType(left, right)
	}
	return nil
}

// getCommonNumericType finds the common type for numeric operations
func getCommonNumericType(left, right semantic.Type) semantic.Type {
	if left.Equals(right) {
		return left
	}

	leftPrim, leftOk := left.(*semantic.PrimitiveType)
	rightPrim, rightOk := right.(*semantic.PrimitiveType)

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

// inferArrayLiteralType infers array literal types
func inferArrayLiteralType(r *analyzer.AnalyzerNode, e *ast.ArrayLiteralExpr, cm *ctx.Module) semantic.Type {
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
	elementType := inferTypeFromExpression(r, e.Elements[0], cm)
	if elementType == nil {
		return nil
	}

	// Verify all elements are compatible
	for _, element := range e.Elements[1:] {
		elemType := inferTypeFromExpression(r, element, cm)
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

	return &semantic.ArrayType{ElementType: elementType, Name: types.ARRAY}
}

// inferIndexableType infers types for array/map indexing
func inferIndexableType(r *analyzer.AnalyzerNode, e *ast.IndexableExpr, cm *ctx.Module) semantic.Type {
	indexableType := inferTypeFromExpression(r, *e.Indexable, cm)
	if indexableType == nil {
		return nil
	}

	// Check if it's an array
	if arrayType, ok := indexableType.(*semantic.ArrayType); ok {
		indexType := inferTypeFromExpression(r, *e.Index, cm)
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
