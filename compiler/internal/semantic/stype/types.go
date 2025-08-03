package stype

import (
	"fmt"
	"sort"
	"strings"

	"ferret/compiler/internal/types"
)

// Type represents a semantic type in the type system (without location information)
type Type interface {
	TypeName() types.TYPE_NAME
	String() string
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
