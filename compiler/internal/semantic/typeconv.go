package semantic

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/types"
)

// ASTToSemanticType converts an AST DataType to a semantic Type
func ASTToSemanticType(astType ast.DataType) Type {
	if astType == nil {
		return nil
	}

	switch t := astType.(type) {
	case *ast.IntType:
		return &PrimitiveType{Name: t.TypeName}
	case *ast.FloatType:
		return &PrimitiveType{Name: t.TypeName}
	case *ast.StringType:
		return &PrimitiveType{Name: t.TypeName}
	case *ast.BoolType:
		return &PrimitiveType{Name: t.TypeName}
	case *ast.ByteType:
		return &PrimitiveType{Name: t.TypeName}
	case *ast.UserDefinedType:
		return &UserType{Name: t.TypeName}
	case *ast.ArrayType:
		elementType := ASTToSemanticType(t.ElementType)
		return &ArrayType{
			ElementType: elementType,
			Name:        t.TypeName,
		}
	case *ast.StructType:
		fields := make(map[string]Type)
		for _, field := range t.Fields {
			if field.FieldType != nil {
				fieldName := field.FieldIdentifier.Name
				fieldType := ASTToSemanticType(field.FieldType)
				fields[fieldName] = fieldType
			}
		}
		return &StructType{
			Name:   t.TypeName,
			Fields: fields,
		}
	case *ast.FunctionType:
		var params []Type
		for _, param := range t.Parameters {
			params = append(params, ASTToSemanticType(param))
		}
		var returns []Type
		for _, ret := range t.ReturnTypes {
			returns = append(returns, ASTToSemanticType(ret))
		}
		return &FunctionType{
			Parameters:  params,
			ReturnTypes: returns,
			Name:        t.TypeName,
		}
	default:
		// For unknown AST types, create a user type
		return &UserType{Name: astType.Type()}
	}
}

// CreatePrimitiveType creates a semantic primitive type
func CreatePrimitiveType(typeName types.TYPE_NAME) Type {
	return &PrimitiveType{Name: typeName}
}

// CreateUserType creates a semantic user-defined type
func CreateUserType(typeName types.TYPE_NAME) Type {
	return &UserType{Name: typeName}
}

// CreateStructType creates a semantic struct type
func CreateStructType(typeName types.TYPE_NAME, fields map[string]Type) Type {
	return &StructType{
		Name:   typeName,
		Fields: fields,
	}
}

// CreateArrayType creates a semantic array type
func CreateArrayType(elementType Type) Type {
	return &ArrayType{
		ElementType: elementType,
		Name:        types.ARRAY,
	}
}

// CreateFunctionType creates a semantic function type
func CreateFunctionType(params []Type, returns []Type) Type {
	return &FunctionType{
		Parameters:  params,
		ReturnTypes: returns,
		Name:        types.FUNCTION,
	}
}

// IsAssignableFrom checks if one type can be assigned from another
func IsAssignableFrom(target, source Type) bool {
	// Same type
	if target.Equals(source) {
		return true
	}

	// Handle user types that are aliases
	if userTarget, ok := target.(*UserType); ok {
		if userTarget.Definition != nil {
			return IsAssignableFrom(userTarget.Definition, source)
		}
	}

	if userSource, ok := source.(*UserType); ok {
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
func isNumericPromotion(target, source Type) bool {
	targetPrim, targetOk := target.(*PrimitiveType)
	sourcePrim, sourceOk := source.(*PrimitiveType)

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
func isArrayCompatible(target, source Type) bool {
	targetArray, targetOk := target.(*ArrayType)
	sourceArray, sourceOk := source.(*ArrayType)

	if !targetOk || !sourceOk {
		return false
	}

	// Arrays are compatible if their element types are assignable
	return IsAssignableFrom(targetArray.ElementType, sourceArray.ElementType)
}

// isFunctionCompatible checks if functions are compatible
func isFunctionCompatible(target, source Type) bool {
	targetFunc, targetOk := target.(*FunctionType)
	sourceFunc, sourceOk := source.(*FunctionType)

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
func isStructCompatible(target, source Type) bool {
	targetStruct, targetOk := target.(*StructType)
	sourceStruct, sourceOk := source.(*StructType)

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

// GetCommonType finds the common type between two types for operations
func GetCommonType(left, right Type) Type {
	// If types are the same, return that type
	if left.Equals(right) {
		return left
	}

	leftPrim, leftOk := left.(*PrimitiveType)
	rightPrim, rightOk := right.(*PrimitiveType)

	if !leftOk || !rightOk {
		return nil // Non-primitive types don't have common types
	}

	leftName := leftPrim.Name
	rightName := rightPrim.Name

	// Numeric type hierarchy for finding common types
	numericHierarchy := map[types.TYPE_NAME]int{
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
func CanImplicitlyConvert(target, source Type) bool {
	return IsAssignableFrom(target, source)
}

// CanExplicitlyConvert checks if source can be explicitly converted to target
func CanExplicitlyConvert(target, source Type) bool {
	// Allow implicit conversions
	if CanImplicitlyConvert(target, source) {
		return true
	}

	targetPrim, targetOk := target.(*PrimitiveType)
	sourcePrim, sourceOk := source.(*PrimitiveType)

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
