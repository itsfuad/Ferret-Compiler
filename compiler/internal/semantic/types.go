package semantic

import (
	"fmt"
	"sort"
	"strings"

	"compiler/internal/frontend/ast"
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
	Parameters  []Type
	ReturnTypes []Type
	Name        types.TYPE_NAME
}

func (f *FunctionType) TypeName() types.TYPE_NAME {
	return f.Name
}

func (f *FunctionType) String() string {
	var paramStrs []string
	for _, param := range f.Parameters {
		paramStrs = append(paramStrs, param.String())
	}

	var returnStrs []string
	for _, ret := range f.ReturnTypes {
		returnStrs = append(returnStrs, ret.String())
	}

	paramStr := strings.Join(paramStrs, ", ")
	returnStr := strings.Join(returnStrs, ", ")

	if len(f.ReturnTypes) == 0 {
		return fmt.Sprintf("fn(%s)", paramStr)
	}
	return fmt.Sprintf("fn(%s) -> %s", paramStr, returnStr)
}

func (f *FunctionType) Equals(other Type) bool {
	if otherFunc, ok := other.(*FunctionType); ok {
		if len(f.Parameters) != len(otherFunc.Parameters) ||
			len(f.ReturnTypes) != len(otherFunc.ReturnTypes) {
			return false
		}

		for i, param := range f.Parameters {
			if !param.Equals(otherFunc.Parameters[i]) {
				return false
			}
		}

		for i, ret := range f.ReturnTypes {
			if !ret.Equals(otherFunc.ReturnTypes[i]) {
				return false
			}
		}

		return true
	}
	return false
}

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

// IsStringType checks if a type is string
func IsStringType(t Type) bool {
	if prim, ok := t.(*PrimitiveType); ok {
		return prim.Name == types.STRING
	}
	return false
}

// IsBoolType checks if a type is boolean
func IsBoolType(t Type) bool {
	if prim, ok := t.(*PrimitiveType); ok {
		return prim.Name == types.BOOL
	}
	return false
}

// IsNumericType checks if a type is numeric
func IsNumericType(t Type) bool {
	if prim, ok := t.(*PrimitiveType); ok {
		return IsNumericTypeName(prim.Name)
	}
	return false
}

// IsIntegerType checks if a type is an integer type
func IsIntegerType(t Type) bool {
	if prim, ok := t.(*PrimitiveType); ok {
		return IsIntegerTypeName(prim.Name)
	}
	return false
}

// IsNumericTypeName checks if a type name is numeric
func IsNumericTypeName(typeName types.TYPE_NAME) bool {
	switch typeName {
	case types.INT8, types.INT16, types.INT32, types.INT64,
		types.UINT8, types.UINT16, types.UINT32, types.UINT64,
		types.FLOAT32, types.FLOAT64, types.BYTE:
		return true
	default:
		return false
	}
}

// IsIntegerTypeName checks if a type name is an integer type
func IsIntegerTypeName(typeName types.TYPE_NAME) bool {
	switch typeName {
	case types.INT8, types.INT16, types.INT32, types.INT64,
		types.UINT8, types.UINT16, types.UINT32, types.UINT64, types.BYTE:
		return true
	default:
		return false
	}
}
