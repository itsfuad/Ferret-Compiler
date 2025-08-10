package stype

import (
	"ferret/internal/types"
	"fmt"
	"sort"
	"strings"
)

// Type represents a semantic type in the type system (without location information)
type Type interface {
	String() string
}

// PrimitiveType represents built-in primitive types (int, string, bool, etc.)
type PrimitiveType struct {
	TypeName types.TYPE_NAME
}

func (p *PrimitiveType) String() string {
	return p.TypeName.String()
}

// UserType represents user-defined types and type aliases
type UserType struct {
	Name       string
	Definition Type                     // For type aliases, this is the underlying type
	Methods    map[string]*FunctionType // Methods associated with this type
}

func (u *UserType) String() string {
	return u.Name
}

// StructType represents struct types with named fields
type StructType struct {
	Fields map[string]Type
}

func (s *StructType) String() string {
	if len(s.Fields) == 0 {
		return "struct {}"
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
		fieldStrs = append(fieldStrs, fmt.Sprintf("%s: %s", fieldName, fieldType))
	}
	return fmt.Sprintf("struct { %s }", strings.Join(fieldStrs, ", "))
}

// GetFieldType returns the type of a field in the struct, or nil if not found
func (s *StructType) GetFieldType(fieldName string) Type {
	return s.Fields[fieldName]
}

// ArrayType represents array types with element type and size
type ArrayType struct {
	ElementType Type
}

func (a *ArrayType) String() string {
	return fmt.Sprintf("[]%s", a.ElementType)
}

type ParamsType struct {
	Name string
	Type Type
}

// FunctionType represents function types with parameters and return type
type FunctionType struct {
	Parameters []ParamsType
	ReturnType Type // Single return type
}

func (f *FunctionType) String() string {
	var paramStrs []string
	for _, param := range f.Parameters {
		paramStrs = append(paramStrs, fmt.Sprintf("%s: %s", param.Name, param.Type))
	}

	return fmt.Sprintf("fn(%s) -> %s", strings.Join(paramStrs, ", "), f.ReturnType)
}

// InterfaceType represents interface types with method signatures
type InterfaceType struct {
	Methods map[string]*FunctionType // method name -> method signature
}

func (i *InterfaceType) String() string {
	if len(i.Methods) == 0 {
		return "interface {}"
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

	return fmt.Sprintf("interface { %s }", strings.Join(methodStrs, "; "))
}

type Invalid struct {
	// Represents an invalid type, used for error handling
}

func (i *Invalid) String() string {
	return "invalid"
}
