package types

func GetNumberBitSize(kind TYPE_NAME) uint8 {
	switch kind {
	case INT8, UINT8, BYTE:
		return 8
	case INT16, UINT16:
		return 16
	case INT32, UINT32, FLOAT32:
		return 32
	case INT64, UINT64, FLOAT64:
		return 64
	default:
		return 0
	}
}

func IsSigned(kind TYPE_NAME) bool {
	switch kind {
	case INT8, INT16, INT32, INT64:
		return true
	default:
		return false
	}
}

func IsUnsigned(kind TYPE_NAME) bool {
	switch kind {
	case UINT8, UINT16, UINT32, UINT64, BYTE:
		return true
	default:
		return false
	}
}

// IsNumericTypeName checks if a type name is numeric
func IsNumericTypeName(typeName TYPE_NAME) bool {
	return IsIntegerTypeName(typeName) || IsFloatTypeName(typeName)
}

// IsIntegerTypeName checks if a type name is an integer type
func IsIntegerTypeName(typeName TYPE_NAME) bool {
	switch typeName {
	case INT8, INT16, INT32, INT64,
		UINT8, UINT16, UINT32, UINT64, BYTE:
		return true
	default:
		return false
	}
}

func IsFloatTypeName(typeName TYPE_NAME) bool {
	switch typeName {
	case FLOAT32, FLOAT64:
		return true
	default:
		return false
	}
}
