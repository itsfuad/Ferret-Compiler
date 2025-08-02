package stype

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
		//source and target user types must have the same name and same fields with same types
		return u.Name == otherUser.Name && u.Definition.Equals(otherUser.Definition)
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

// ArrayType represents array types with element type and size
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

// FunctionType represents function types with parameters and return type
type FunctionType struct {
	Parameters []Type
	ReturnType Type // Single return type (multiple returns removed)
	Name       types.TYPE_NAME
}

func (f *FunctionType) TypeName() types.TYPE_NAME {
	return f.Name
}

func (f *FunctionType) String() string {
	var paramStrs []string
	for _, param := range f.Parameters {
		paramStrs = append(paramStrs, param.String())
	}

	return fmt.Sprintf("fn(%s) -> %s", strings.Join(paramStrs, ", "), f.ReturnType.String())
}

func (f *FunctionType) Equals(other Type) bool {
	if otherFunc, ok := other.(*FunctionType); ok {
		// Check parameter count
		if len(f.Parameters) != len(otherFunc.Parameters) {
			return false
		}

		// Check each parameter type
		for i, param := range f.Parameters {
			if !param.Equals(otherFunc.Parameters[i]) {
				return false
			}
		}

		// Check return type
		return f.ReturnType.Equals(otherFunc.ReturnType)
	}
	return false
}

// InterfaceType represents interface types with method signatures
type InterfaceType struct {
	Name    types.TYPE_NAME
	Methods map[string]*FunctionType // method name -> method signature
}

func (i *InterfaceType) TypeName() types.TYPE_NAME {
	return i.Name
}

func (i *InterfaceType) String() string {
	if len(i.Methods) == 0 {
		return fmt.Sprintf("interface %s {}", i.Name)
	}

	// Collect method names and sort them for consistent output
	var methodNames []string
	for methodName := range i.Methods {
		methodNames = append(methodNames, methodName)
	}
	sort.Strings(methodNames)

	// Build method strings in alphabetical order
	var methodStrs []string
	for _, methodName := range methodNames {
		methodType := i.Methods[methodName]
		methodStrs = append(methodStrs, fmt.Sprintf("%s%s", methodName, methodType.String()[2:])) // Remove "fn" prefix
	}

	return fmt.Sprintf("interface %s { %s }", i.Name, strings.Join(methodStrs, "; "))
}

func (i *InterfaceType) Equals(other Type) bool {
	if otherInterface, ok := other.(*InterfaceType); ok {
		return i.Name == otherInterface.Name
	}
	return false
}

// GetMethod returns the method signature for a given method name, or nil if not found
func (i *InterfaceType) GetMethod(methodName string) *FunctionType {
	return i.Methods[methodName]
}
