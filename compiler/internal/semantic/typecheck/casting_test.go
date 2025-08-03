package typecheck

import (
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/types"
	"fmt"
	"testing"
)

// TestImplicitCast tests numeric type promotions
func TestImplicitCast(t *testing.T) {
	int8Type := &stype.PrimitiveType{Name: types.INT8}
	int16Type := &stype.PrimitiveType{Name: types.INT16}
	int32Type := &stype.PrimitiveType{Name: types.INT32}
	float32Type := &stype.PrimitiveType{Name: types.FLOAT32}
	float64Type := &stype.PrimitiveType{Name: types.FLOAT64}
	// user type
	userType1 := &stype.UserType{Name: "UserType1", Definition: &stype.StructType{
		Name: "UserType1",
		Fields: map[string]stype.Type{
			"field1": int8Type,
			"field2": int16Type,
		},
	}}

	userType2 := &stype.UserType{Name: "UserType2", Definition: &stype.StructType{
		Name: "UserType2",
		Fields: map[string]stype.Type{
			"field1": int16Type,
			"field2": int32Type,
		},
	}}

	userType3 := &stype.UserType{Name: "UserType3", Definition: &stype.StructType{
		Name: "UserType3",
		Fields: map[string]stype.Type{
			"field1": float32Type,
			"field2": float64Type,
			"field3": int32Type,
		},
	}}

	//loop through tests
	var tests = []struct {
		target   stype.Type
		source   stype.Type
		expected bool
	}{
		{int8Type, int16Type, false},     // int8 cannot be assigned
		{int16Type, int8Type, true},      // int16 can be assigned from int8
		{int32Type, int16Type, true},     // int32 can be assigned from int16
		{float32Type, int32Type, false},  // float32 cannot be assigned from int32
		{float64Type, float32Type, true}, // float64 can be assigned from float32
		{userType1, userType2, false},    // UserType1 cannot be assigned to UserType2
		{userType2, userType1, true},     // UserType2 can be assigned from UserType1
		{userType3, userType1, false},    // UserType3 cannot be assigned to UserType1
		{userType1, userType3, false},    // UserType1 can be assigned from UserType3
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s->%s", tt.source.TypeName(), tt.target.TypeName()), func(t *testing.T) {
			result := isImplicitCastable(tt.target, tt.source)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExplicitCast(t *testing.T) {
	int8Type := &stype.PrimitiveType{Name: types.INT8}
	int16Type := &stype.PrimitiveType{Name: types.INT16}
	byteType := &stype.PrimitiveType{Name: types.BYTE}
	int32Type := &stype.PrimitiveType{Name: types.INT32}
	float32Type := &stype.PrimitiveType{Name: types.FLOAT32}

	var tests = []struct {
		target   stype.Type
		source   stype.Type
		expected bool
	}{
		{int8Type, int16Type, true},    // int8 cannot be assigned from int16
		{int16Type, int8Type, true},    // int16 can be assigned from int8
		{int32Type, int16Type, true},   // int32 can be assigned from int16
		{float32Type, int32Type, true}, // float32 can be assigned from int32
		{int32Type, float32Type, true}, // int32 cannot be assigned from float32
		{byteType, int8Type, true},     // byte can be assigned from int8
		{int8Type, byteType, true},     // int8 cannot be assigned from byte
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s->%s", tt.source.TypeName(), tt.target.TypeName()), func(t *testing.T) {
			result, _ := isPrimitiveExplicitCastable(tt.target, tt.source)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
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
	if !isImplicitCastable(int64ArrayType, int32ArrayType) {
		t.Error("int32[] should be assignable to int64[] (element promotion)")
	}

	if isImplicitCastable(int32ArrayType, int64ArrayType) {
		t.Error("int64[] should NOT be assignable to int32[] (no element demotion)")
	}
}
