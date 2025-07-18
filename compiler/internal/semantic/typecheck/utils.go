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

// Numeric type hierarchy for finding common types
var numericHierarchy map[types.TYPE_NAME]int = map[types.TYPE_NAME]int{
	types.INT8:    1,
	types.UINT8:   1,
	types.BYTE:    1,
	types.INT16:   2,
	types.UINT16:  2,
	types.INT32:   3,
	types.UINT32:  3,
	types.INT64:   4,
	types.UINT64:  4,
	types.FLOAT32: 5,
	types.FLOAT64: 6,
}

// IsAssignableFrom checks if one type can be assigned from another
func IsAssignableFrom(target, source semantic.Type) bool {
	// Same type
	if target.Equals(source) {
		return true
	}

	// Handle user types that are aliases
	if userTarget, ok := target.(*semantic.UserType); ok {
		if userTarget.Definition != nil {
			return IsAssignableFrom(userTarget.Definition, source)
		}
	}

	if userSource, ok := source.(*semantic.UserType); ok {
		if userSource.Definition != nil {
			return IsAssignableFrom(target, userSource.Definition)
		}
	}

	// Numeric type promotions
	if isNumericPromotion(target, source) {
		return true
	}

	// Array type compatibility
	if isArrayCompatible(target, source) {
		return true
	}

	// Function type compatibility
	if isFunctionCompatible(target, source) {
		return true
	}

	// Struct type compatibility (structural typing)
	if isStructCompatible(target, source) {
		return true
	}

	return false
}

// isNumericPromotion checks if source type can be promoted to target type
func isNumericPromotion(target, source semantic.Type) bool {
	targetPrim, targetOk := target.(*semantic.PrimitiveType)
	sourcePrim, sourceOk := source.(*semantic.PrimitiveType)

	if !targetOk || !sourceOk {
		return false
	}

	targetName := targetPrim.Name
	sourceName := sourcePrim.Name

	// Integer promotions: smaller -> larger
	integerPromotions := map[types.TYPE_NAME][]types.TYPE_NAME{
		types.INT16:  {types.INT8},
		types.INT32:  {types.INT8, types.INT16},
		types.INT64:  {types.INT8, types.INT16, types.INT32},
		types.UINT16: {types.UINT8, types.BYTE},
		types.UINT32: {types.UINT8, types.UINT16, types.BYTE},
		types.UINT64: {types.UINT8, types.UINT16, types.UINT32, types.BYTE},
	}

	// Float promotions: smaller -> larger, int -> float
	floatPromotions := map[types.TYPE_NAME][]types.TYPE_NAME{
		types.FLOAT32: {types.INT8, types.INT16, types.UINT8, types.UINT16, types.BYTE},
		types.FLOAT64: {types.INT8, types.INT16, types.INT32, types.UINT8, types.UINT16, types.UINT32, types.BYTE, types.FLOAT32},
	}

	// Check integer promotions
	if allowedSources, exists := integerPromotions[targetName]; exists {
		for _, allowedSource := range allowedSources {
			if sourceName == allowedSource {
				return true
			}
		}
	}

	// Check float promotions
	if allowedSources, exists := floatPromotions[targetName]; exists {
		for _, allowedSource := range allowedSources {
			if sourceName == allowedSource {
				return true
			}
		}
	}

	return false
}

// isArrayCompatible checks if arrays are compatible
func isArrayCompatible(target, source semantic.Type) bool {
	targetArray, targetOk := target.(*semantic.ArrayType)
	sourceArray, sourceOk := source.(*semantic.ArrayType)

	if !targetOk || !sourceOk {
		return false
	}

	// Arrays are compatible if their element types are assignable
	return IsAssignableFrom(targetArray.ElementType, sourceArray.ElementType)
}

// isFunctionCompatible checks if functions are compatible
func isFunctionCompatible(target, source semantic.Type) bool {
	targetFunc, targetOk := target.(*semantic.FunctionType)
	sourceFunc, sourceOk := source.(*semantic.FunctionType)

	if !targetOk || !sourceOk {
		return false
	}

	// Parameter count must match
	if len(targetFunc.Parameters) != len(sourceFunc.Parameters) {
		return false
	}

	// Return type count must match
	if len(targetFunc.ReturnTypes) != len(sourceFunc.ReturnTypes) {
		return false
	}

	// Parameters must be contravariant (source params can accept target params)
	for i, targetParam := range targetFunc.Parameters {
		sourceParam := sourceFunc.Parameters[i]
		if !IsAssignableFrom(sourceParam, targetParam) {
			return false
		}
	}

	// Return types must be covariant (target returns can accept source returns)
	for i, targetReturn := range targetFunc.ReturnTypes {
		sourceReturn := sourceFunc.ReturnTypes[i]
		if !IsAssignableFrom(targetReturn, sourceReturn) {
			return false
		}
	}

	return true
}

// isStructCompatible checks if structs are compatible (structural typing)
func isStructCompatible(target, source semantic.Type) bool {
	targetStruct, targetOk := target.(*semantic.StructType)
	sourceStruct, sourceOk := source.(*semantic.StructType)

	if !targetOk || !sourceOk {
		return false
	}

	// Target struct must have all fields that source struct has with compatible types
	for fieldName, sourceFieldType := range sourceStruct.Fields {
		targetFieldType, exists := targetStruct.Fields[fieldName]
		if !exists {
			return false // Target missing field that source has
		}

		if !IsAssignableFrom(targetFieldType, sourceFieldType) {
			return false // Field types not compatible
		}
	}

	return true
}

// getCommonNumericType finds the common type between two types for operations
func getCommonNumericType(left, right semantic.Type) semantic.Type {
	// If types are the same, return that type
	if left.Equals(right) {
		return left
	}

	leftPrim, leftOk := left.(*semantic.PrimitiveType)
	rightPrim, rightOk := right.(*semantic.PrimitiveType)

	if !leftOk || !rightOk {
		return nil // Non-primitive types don't have common types
	}

	leftName := leftPrim.Name
	rightName := rightPrim.Name

	leftLevel, leftExists := numericHierarchy[leftName]
	rightLevel, rightExists := numericHierarchy[rightName]

	if !leftExists || !rightExists {
		return nil // Non-numeric types
	}

	// Return the higher level type
	if leftLevel >= rightLevel {
		return left
	}
	return right
}

// CanImplicitlyConvert checks if source can be implicitly converted to target
func CanImplicitlyConvert(target, source semantic.Type) bool {
	return IsAssignableFrom(target, source)
}

// CanExplicitlyConvert checks if source can be explicitly converted to target
func CanExplicitlyConvert(target, source semantic.Type) bool {
	// Allow implicit conversions
	if CanImplicitlyConvert(target, source) {
		return true
	}

	targetPrim, targetOk := target.(*semantic.PrimitiveType)
	sourcePrim, sourceOk := source.(*semantic.PrimitiveType)

	if !targetOk || !sourceOk {
		return false
	}

	targetName := targetPrim.Name
	sourceName := sourcePrim.Name

	// Allow explicit conversions between numeric types
	numericTypes := map[types.TYPE_NAME]bool{
		types.INT8:    true,
		types.INT16:   true,
		types.INT32:   true,
		types.INT64:   true,
		types.UINT8:   true,
		types.UINT16:  true,
		types.UINT32:  true,
		types.UINT64:  true,
		types.FLOAT32: true,
		types.FLOAT64: true,
		types.BYTE:    true,
	}

	if numericTypes[targetName] && numericTypes[sourceName] {
		return true
	}

	return false
}


// inferExpressionType infers the type of an expression
func inferExpressionType(r *analyzer.AnalyzerNode, expr ast.Expression, cm *ctx.Module) semantic.Type {

	if expr == nil {
		return nil
	}

	var resultType semantic.Type

	switch e := expr.(type) {
	// Primitive types
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
	// Expressions
	case *ast.IdentifierExpr:
		resultType = inferIdentifierType(r, e, cm)
	case *ast.BinaryExpr:
		resultType = inferBinaryExprType(r, e, cm)
	case *ast.ArrayLiteralExpr:
		resultType = inferArrayLiteralType(r, e, cm)
	case *ast.IndexableExpr:
		resultType = inferIndexableType(r, e, cm)
	default:
		resultType = nil
	}

	logInferredType(r, expr, resultType)

	return resultType
}

// inferIdentifierType infers the type of an identifier expression
func inferIdentifierType(_ *analyzer.AnalyzerNode, e *ast.IdentifierExpr, currentModule *ctx.Module) semantic.Type {
	sym, found := currentModule.SymbolTable.Lookup(e.Name)
	if found {
		return sym.Type
	}
	return nil
}


// inferBinaryExprType infers the type of a binary expression
func inferBinaryExprType(r *analyzer.AnalyzerNode, e *ast.BinaryExpr, currentModule *ctx.Module) semantic.Type {

	leftType := inferExpressionType(r, *e.Left, currentModule)
	rightType := inferExpressionType(r, *e.Right, currentModule)

	if leftType == nil || rightType == nil {
		return nil
	}

	resultType := inferBinaryOperationType(e.Operator.Value, leftType, rightType)
	if resultType == nil {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"invalid binary operation: "+leftType.String()+" "+e.Operator.Value+" "+rightType.String(),
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
	}
	return resultType
}

// inferBinaryOperationType infers the result type of a binary operation
func inferBinaryOperationType(operator string, leftType, rightType semantic.Type) semantic.Type {
	switch operator {
	case "+", "-", "*", "/", "%":
		return inferArithmeticOperationType(operator, leftType, rightType)
	case "==", "!=", "<", "<=", ">", ">=":
		return inferComparisonOperationType(leftType, rightType)
	case "&&", "||":
		return inferLogicalOperationType(leftType, rightType)
	case "&", "|", "^", "<<", ">>":
		return inferBitwiseOperationType(leftType, rightType)
	default:
		return nil
	}
}


// inferArithmeticOperationType handles arithmetic operations
func inferArithmeticOperationType(operator string, leftType, rightType semantic.Type) semantic.Type {
	// Arithmetic operations - return common numeric type
	commonType := getCommonNumericType(leftType, rightType)
	if commonType != nil {
		return commonType
	}

	// String concatenation with + (only str + str)
	if operator == "+" {
		leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
		rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

		if leftOk && rightOk &&
			leftPrim.Name == types.STRING && rightPrim.Name == types.STRING {
			return &semantic.PrimitiveType{Name: types.STRING}
		}
	}

	// If we reach here, the operation is invalid
	return nil
}

// inferComparisonOperationType handles comparison operations
func inferComparisonOperationType(leftType, rightType semantic.Type) semantic.Type {
	// Check if types are comparable
	if IsAssignableFrom(leftType, rightType) ||
		IsAssignableFrom(rightType, leftType) ||
		getCommonNumericType(leftType, rightType) != nil {
		return &semantic.PrimitiveType{Name: types.BOOL}
	}
	return nil
}

// inferLogicalOperationType handles logical operations
func inferLogicalOperationType(leftType, rightType semantic.Type) semantic.Type {
	leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
	rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

	if leftOk && rightOk &&
		leftPrim.Name == types.BOOL && rightPrim.Name == types.BOOL {
		return &semantic.PrimitiveType{Name: types.BOOL}
	}
	return nil
}

// inferBitwiseOperationType handles bitwise operations
func inferBitwiseOperationType(leftType, rightType semantic.Type) semantic.Type {
	leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
	rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

	if leftOk && rightOk && isIntegerType(leftPrim.Name) && isIntegerType(rightPrim.Name) {
		commonType := getCommonNumericType(leftType, rightType)
		if commonType != nil {
			return commonType
		}
	}
	return nil
}

// resolveType resolves a type alias to its underlying type
func resolveType(r *analyzer.AnalyzerNode, t semantic.Type) semantic.Type {
	//recursively call this function until we reach a non-alias type
	if userType, ok := t.(*semantic.UserType); ok {
		if userType.Definition != nil {
			return resolveType(r, userType.Definition)
		}
		return t // If no definition, return the alias itself
	}
	return t
}

// inferArrayLiteralType infers the type of an array literal expression
func inferArrayLiteralType(r *analyzer.AnalyzerNode, e *ast.ArrayLiteralExpr, currentModule *ctx.Module) semantic.Type {
	if len(e.Elements) == 0 {
		// Empty array - cannot infer type
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"cannot infer array type from empty array literal",
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}

	// Infer type from first element
	firstElementType := inferExpressionType(r, e.Elements[0], currentModule)
	if firstElementType == nil {
		return nil
	}

	// Check that all elements have compatible types
	commonType := firstElementType
	for _, element := range e.Elements {
		elementType := inferExpressionType(r, element, currentModule)
		if elementType == nil {
			continue
		}

		// Try to find a common type
		newCommonType := getCommonNumericType(commonType, elementType)
		if newCommonType == nil {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				element.Loc(),
				"array element type mismatch: cannot use "+elementType.String()+" in array of "+commonType.String(),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
			return nil
		}
		commonType = newCommonType
	}

	// Create array type with the common element type
	return &semantic.ArrayType{ElementType: commonType, Name: types.ARRAY}
}

// inferIndexableType infers the type of an array/map indexing expression
func inferIndexableType(r *analyzer.AnalyzerNode, e *ast.IndexableExpr, currentModule *ctx.Module) semantic.Type {
	// Get the type of the indexable expression
	indexableType := inferExpressionType(r, *e.Indexable, currentModule)
	if indexableType == nil {
		return nil
	}

	// Check if it's an array type
	if arrayType, ok := indexableType.(*semantic.ArrayType); ok {
		// Verify the index is an integer type
		indexType := inferExpressionType(r, *e.Index, currentModule)
		if indexType == nil {
			return nil
		}

		// Check if index type is an integer
		if !isIntegerTypeForIndexing(indexType) {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				(*e.Index).Loc(),
				"array index must be an integer type, got "+indexType.String(),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
			return nil
		}

		// Return the element type of the array
		return arrayType.ElementType
	}

	// If not an array, report error
	r.Ctx.Reports.Add(
		r.Program.FullPath,
		(*e.Indexable).Loc(),
		"cannot index non-array type "+indexableType.String(),
		report.TYPECHECK_PHASE,
	).SetLevel(report.SEMANTIC_ERROR)
	return nil
}

// isIntegerTypeForIndexing checks if a type can be used as an array index
func isIntegerTypeForIndexing(t semantic.Type) bool {
	if primType, ok := t.(*semantic.PrimitiveType); ok {
		return isIntegerType(primType.Name)
	}
	return false
}

// isIntegerType checks if a type is an integer type
func isIntegerType(typeName types.TYPE_NAME) bool {
	switch typeName {
	case types.INT8, types.INT16, types.INT32, types.INT64,
		types.UINT8, types.UINT16, types.UINT32, types.UINT64, types.BYTE:
		return true
	default:
		return false
	}
}

// logInferredType logs the inferred type for debugging
func logInferredType(r *analyzer.AnalyzerNode, expr ast.Expression, resultType semantic.Type) {
	if r.Debug {
		if resultType == nil {
			colors.YELLOW.Printf("Inferred type for expression '%v': <nil>\n", expr)
		} else {
			colors.YELLOW.Printf("Inferred type for expression '%v': %s\n", expr, resultType.String())
		}
	}
}