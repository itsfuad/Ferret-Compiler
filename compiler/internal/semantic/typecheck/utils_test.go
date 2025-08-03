package typecheck

import (
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/types"
	"testing"
)

// TestNumericTypeHierarchy tests numeric type promotions
func TestNumericTypeHierarchy(t *testing.T) {
	int8Type := &stype.PrimitiveType{Name: types.INT8}
	int16Type := &stype.PrimitiveType{Name: types.INT16}
	int32Type := &stype.PrimitiveType{Name: types.INT32}
	float32Type := &stype.PrimitiveType{Name: types.FLOAT32}
	float64Type := &stype.PrimitiveType{Name: types.FLOAT64}

	// Test integer promotions
	if !IsAssignableFrom(int16Type, int8Type) {
		t.Error("int8 should promote to int16")
	}

	if !IsAssignableFrom(int32Type, int16Type) {
		t.Error("int16 should promote to int32")
	}

	// Test float promotions
	if !IsAssignableFrom(float32Type, int8Type) {
		t.Error("int8 should promote to float32")
	}

	if !IsAssignableFrom(float64Type, float32Type) {
		t.Error("float32 should promote to float64")
	}

	if !IsAssignableFrom(float64Type, int32Type) {
		t.Error("int32 should promote to float64")
	}
}

// TestCommonNumericType tests finding common types for operations
func TestCommonNumericType(t *testing.T) {
	int32Type := &stype.PrimitiveType{Name: types.INT32}
	int64Type := &stype.PrimitiveType{Name: types.INT64}
	float32Type := &stype.PrimitiveType{Name: types.FLOAT32}
	float64Type := &stype.PrimitiveType{Name: types.FLOAT64}

	// Test same types
	result := getCommonNumericType(int32Type, int32Type)
	if result.TypeName() != types.INT32 {
		t.Error("common type of same types should be that type")
	}

	// Test int32 + int64 -> int64
	result = getCommonNumericType(int32Type, int64Type)
	if result.TypeName() != types.INT64 {
		t.Error("common type of int32 and int64 should be int64")
	}

	// Test int32 + float32 -> float32
	result = getCommonNumericType(int32Type, float32Type)
	if result.TypeName() != types.FLOAT32 {
		t.Error("common type of int32 and float32 should be float32")
	}

	// Test float32 + float64 -> float64
	result = getCommonNumericType(float32Type, float64Type)
	if result.TypeName() != types.FLOAT64 {
		t.Error("common type of float32 and float64 should be float64")
	}
}

// TestArrayCompatibility tests array type compatibility
func TestArrayCompatibility(t *testing.T) {
	int32Type := &stype.PrimitiveType{Name: types.INT32}
	int64Type := &stype.PrimitiveType{Name: types.INT64}

	int32ArrayType := &stype.ArrayType{ElementType: int32Type, Name: types.ARRAY}
	int64ArrayType := &stype.ArrayType{ElementType: int64Type, Name: types.ARRAY}

	// Arrays are compatible if element types are assignable
	if !IsAssignableFrom(int64ArrayType, int32ArrayType) {
		t.Error("int32[] should be assignable to int64[] (element promotion)")
	}

	if IsAssignableFrom(int32ArrayType, int64ArrayType) {
		t.Error("int64[] should NOT be assignable to int32[] (no element demotion)")
	}
}
