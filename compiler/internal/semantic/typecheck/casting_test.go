package typecheck

import (
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/types"
	"fmt"
	"testing"
)

// TestImplicitCast tests numeric type promotions
func TestImplicitCast(t *testing.T) {
	int8Type := &stype.PrimitiveType{TypeName: types.INT8}
	int16Type := &stype.PrimitiveType{TypeName: types.INT16}
	int32Type := &stype.PrimitiveType{TypeName: types.INT32}
	float32Type := &stype.PrimitiveType{TypeName: types.FLOAT32}
	float64Type := &stype.PrimitiveType{TypeName: types.FLOAT64}
	// user type
	userType1 := &stype.UserType{Name: "UserType1", Definition: &stype.StructType{
		Fields: map[string]stype.Type{
			"field1": int8Type,
			"field2": int16Type,
		},
	}}

	userType2 := &stype.UserType{Name: "UserType2", Definition: &stype.StructType{
		Fields: map[string]stype.Type{
			"field1": int16Type,
			"field2": int32Type,
		},
	}}

	userType3 := &stype.UserType{Name: "UserType3", Definition: &stype.StructType{
		Fields: map[string]stype.Type{
			"field1": float32Type,
			"field2": float64Type,
			"field3": int32Type,
		},
	}}

	userType4 := &stype.UserType{Name: "UserType4", Definition: &stype.StructType{
		Fields: map[string]stype.Type{
			"field1": float32Type,
			"field2": float32Type,
			"field3": int32Type,
			"field4": &stype.ArrayType{ElementType: int32Type},
		},
	}}

	//loop through tests
	var tests = []struct {
		source   stype.Type
		target   stype.Type
		expected bool
	}{
		{int16Type, int8Type, false},
		{int8Type, int16Type, true},
		{int16Type, int32Type, true},
		{float32Type, int32Type, false},
		{float32Type, float64Type, true},
		{userType2, userType1, false},
		{userType1, userType2, true},
		{userType3, userType1, false},
		{userType1, userType3, false},
		{userType4, userType3, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s->%s : %v", tt.source, tt.target, tt.expected), func(t *testing.T) {
			result, _ := isImplicitCastable(tt.target, tt.source)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExplicitCast(t *testing.T) {
	int8Type := &stype.PrimitiveType{TypeName: types.INT8}
	int16Type := &stype.PrimitiveType{TypeName: types.INT16}
	byteType := &stype.PrimitiveType{TypeName: types.BYTE}
	int32Type := &stype.PrimitiveType{TypeName: types.INT32}
	float32Type := &stype.PrimitiveType{TypeName: types.FLOAT32}

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
		t.Run(fmt.Sprintf("%s->%s", tt.source, tt.target), func(t *testing.T) {
			result, _ := isPrimitiveExplicitCastable(tt.target, tt.source)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCommonNumericType tests finding common types for operations
func TestCommonNumericType(t *testing.T) {
	int32Type := &stype.PrimitiveType{TypeName: types.INT32}
	int64Type := &stype.PrimitiveType{TypeName: types.INT64}
	float32Type := &stype.PrimitiveType{TypeName: types.FLOAT32}
	float64Type := &stype.PrimitiveType{TypeName: types.FLOAT64}

	// Test same types
	result, ok := getCommonNumericType(int32Type, int32Type).(*stype.PrimitiveType)
	if !ok || result.TypeName != types.INT32 {
		t.Error("common type of same types should be that type")
	}

	// Test int32 + int64 -> int64
	result, ok = getCommonNumericType(int32Type, int64Type).(*stype.PrimitiveType)
	if !ok || result.TypeName != types.INT64 {
		t.Error("common type of int32 and int64 should be int64")
	}

	// Test int32 + float32 -> float32
	result, ok = getCommonNumericType(int32Type, float32Type).(*stype.PrimitiveType)
	if !ok || result.TypeName != types.FLOAT32 {
		t.Error("common type of int32 and float32 should be float32")
	}

	// Test float32 + float64 -> float64
	result, ok = getCommonNumericType(float32Type, float64Type).(*stype.PrimitiveType)
	if !ok || result.TypeName != types.FLOAT64 {
		t.Error("common type of float32 and float64 should be float64")
	}
}

// TestArrayCompatibility tests array type compatibility
func TestArrayCompatibility(t *testing.T) {
	int32Type := &stype.PrimitiveType{TypeName: types.INT32}
	int64Type := &stype.PrimitiveType{TypeName: types.INT64}

	int32ArrayType := &stype.ArrayType{ElementType: int32Type}
	int64ArrayType := &stype.ArrayType{ElementType: int64Type}

	// Arrays are compatible if element types are assignable
	if ok, _ := isImplicitCastable(int64ArrayType, int32ArrayType); !ok {
		t.Error("int32[] should be assignable to int64[] (element promotion)")
	}

	if ok, _ := isImplicitCastable(int32ArrayType, int64ArrayType); ok {
		t.Error("int64[] should NOT be assignable to int32[] (no element demotion)")
	}
}
