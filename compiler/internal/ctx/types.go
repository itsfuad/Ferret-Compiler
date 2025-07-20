package ctx

import (
	"fmt"
	"sort"
	"strings"

	"compiler/internal/types"
)

// Type represents a semantic type in the type system (without location information)
type Type interface {
	TypeName() types.TYPE_NAME
	String() string
	Equals(other Type) bool
}

// PrimitiveType represents built-in primitive types (int, string, bool, etc.)
type PrimitiveType struct {
	Name types.TYPE_NAME
}

func (p *PrimitiveType) TypeName() types.TYPE_NAME {
	return p.Name
}

func (p *PrimitiveType) String() string {
	return string(p.Name)
}

func (p *PrimitiveType) Equals(other Type) bool {
	if otherPrim, ok := other.(*PrimitiveType); ok {
		return p.Name == otherPrim.Name
	}
	return false
}

// UserType represents user-defined types and type aliases
type UserType struct {
	Name       types.TYPE_NAME
	Definition Type // For type aliases, this is the underlying type
}

func (u *UserType) TypeName() types.TYPE_NAME {
	return u.Name
}

func (u *UserType) String() string {
	return string(u.Name)
}

func (u *UserType) Equals(other Type) bool {
	if otherUser, ok := other.(*UserType); ok {
		return u.Name == otherUser.Name
	}
	return false
}

// StructType represents struct types with named fields
type StructType struct {
	Name   types.TYPE_NAME
	Fields map[string]Type
}

func (s *StructType) TypeName() types.TYPE_NAME {
	return s.Name
}

func (s *StructType) String() string {
	if len(s.Fields) == 0 {
		return fmt.Sprintf("%s {}", s.Name)
	}

	// Collect field names and sort them for consistent output
	var fieldNames []string
	for fieldName := range s.Fields {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	// Build field strings in alphabetical order
	var fieldStrs []string
	for _, fieldName := range fieldNames {
		fieldType := s.Fields[fieldName]
		fieldStrs = append(fieldStrs, fmt.Sprintf("%s: %s", fieldName, fieldType.String()))
	}
	return fmt.Sprintf("%s { %s }", s.Name, strings.Join(fieldStrs, ", "))
}

func (s *StructType) Equals(other Type) bool {
	if otherStruct, ok := other.(*StructType); ok {
		return s.Name == otherStruct.Name
	}
	return false
}

// GetFieldType returns the type of a field in the struct, or nil if not found
func (s *StructType) GetFieldType(fieldName string) Type {
	return s.Fields[fieldName]
}

// HasField checks if the struct has a field with the given name
func (s *StructType) HasField(fieldName string) bool {
	_, exists := s.Fields[fieldName]
	return exists
}

// ArrayType represents array types
type ArrayType struct {
	ElementType Type
	Name        types.TYPE_NAME
}

func (a *ArrayType) TypeName() types.TYPE_NAME {
	return a.Name
}

func (a *ArrayType) String() string {
	return fmt.Sprintf("[]%s", a.ElementType.String())
}

func (a *ArrayType) Equals(other Type) bool {
	if otherArray, ok := other.(*ArrayType); ok {
		return a.ElementType.Equals(otherArray.ElementType)
	}
	return false
}

// FunctionType represents function types
type FunctionType struct {
	Parameters []Type
	ReturnType Type
	Name       types.TYPE_NAME
}

func (f *FunctionType) TypeName() types.TYPE_NAME {
	return f.Name
}

func (f *FunctionType) String() string {
	var params strings.Builder
	params.WriteString("(")
	for i, param := range f.Parameters {
		if i > 0 {
			params.WriteString(", ")
		}
		params.WriteString(param.String())
	}
	params.WriteString(")")

	if f.ReturnType != nil {
		params.WriteString(" -> ")
		params.WriteString(f.ReturnType.String())
	}

	return "fn" + params.String()
}

func (f *FunctionType) Equals(other Type) bool {
	if otherFunc, ok := other.(*FunctionType); ok {
		if len(f.Parameters) != len(otherFunc.Parameters) {
			return false
		}

		// Compare parameters
		for i, param := range f.Parameters {
			if !param.Equals(otherFunc.Parameters[i]) {
				return false
			}
		}

		// Compare return types
		if f.ReturnType == nil && otherFunc.ReturnType == nil {
			return true
		}
		if f.ReturnType == nil || otherFunc.ReturnType == nil {
			return false
		}
		return f.ReturnType.Equals(otherFunc.ReturnType)
	}
	return false
}
