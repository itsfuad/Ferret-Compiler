package typecheck

import (
	"compiler/internal/semantic"
	"compiler/internal/types"
)

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
	commonType := semantic.GetCommonType(leftType, rightType)
	if commonType != nil {
		return commonType
	}

	// String concatenation with + (only str + str)
	if operator == "+" {
		leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
		rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

		if leftOk && rightOk &&
			leftPrim.Name == types.STRING && rightPrim.Name == types.STRING {
			return semantic.CreatePrimitiveType(types.STRING)
		}
	}

	// If we reach here, the operation is invalid
	return nil
}

// inferComparisonOperationType handles comparison operations
func inferComparisonOperationType(leftType, rightType semantic.Type) semantic.Type {
	// Check if types are comparable
	if semantic.IsAssignableFrom(leftType, rightType) ||
		semantic.IsAssignableFrom(rightType, leftType) ||
		semantic.GetCommonType(leftType, rightType) != nil {
		return semantic.CreatePrimitiveType(types.BOOL)
	}
	return nil
}

// inferLogicalOperationType handles logical operations
func inferLogicalOperationType(leftType, rightType semantic.Type) semantic.Type {
	leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
	rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

	if leftOk && rightOk &&
		leftPrim.Name == types.BOOL && rightPrim.Name == types.BOOL {
		return semantic.CreatePrimitiveType(types.BOOL)
	}
	return nil
}

// inferBitwiseOperationType handles bitwise operations
func inferBitwiseOperationType(leftType, rightType semantic.Type) semantic.Type {
	leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
	rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

	if leftOk && rightOk && isIntegerType(leftPrim.Name) && isIntegerType(rightPrim.Name) {
		commonType := semantic.GetCommonType(leftType, rightType)
		if commonType != nil {
			return commonType
		}
	}
	return nil
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
