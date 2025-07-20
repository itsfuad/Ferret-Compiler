package ctx

import (
	"fmt"

	"compiler/internal/frontend/ast"
	"compiler/internal/types"
)

// DeriveSemanticType converts an AST DataType to a semantic Type
func DeriveSemanticType(astType ast.DataType, module *Module) (Type, error) {

	if astType == nil {
		return nil, fmt.Errorf("nil AST type provided")
	}

	switch t := astType.(type) {
	case *ast.IntType, *ast.FloatType, *ast.StringType, *ast.BoolType, *ast.ByteType:
		return derivePrimitiveTypeFromAst(t)
	case *ast.ArrayType:
		return deriveSemanticArrayType(t, module)
	case *ast.StructType:
		return deriveSemanticStructFromAst(t, module)
	case *ast.FunctionType:
		return deriveSemanticFunctionType(t, module)
	case *ast.TypeScopeResolution:
		return resolveTypeInImportedModule(t, module)
	}
	//user-defined types or aliases
	if tt, ok := module.SymbolTable.Lookup(string(astType.Type())); ok {
		if tt.Type != nil {
			return tt.Type, nil
		}
		return &UserType{Name: tt.Type.TypeName(), Definition: nil}, nil // No definition available
	}
	return nil, fmt.Errorf("type '%s' not found in symbol table", astType.Type())
}

func derivePrimitiveTypeFromAst(astType ast.DataType) (*PrimitiveType, error) {
	return &PrimitiveType{
		Name: astType.Type(),
	}, nil
}

func resolveTypeInImportedModule(res *ast.TypeScopeResolution, module *Module) (Type, error) {
	// Handle type scope resolution (e.g., module::TypeName)
	moduleName := res.Module.Name
	typeName := res.TypeNode.Type()
	symbolTable, found := module.SymbolTable.Imports[moduleName]
	if !found {
		return nil, fmt.Errorf("module '%s' is not imported", moduleName)
	}
	// Look up the type in the imported module's symbol table
	if symbol, ok := symbolTable.Lookup(string(typeName)); ok {
		if symbol.Type != nil {
			return symbol.Type, nil
		}
		return &UserType{Name: symbol.Type.TypeName(), Definition: nil}, nil // No definition available
	}
	return nil, fmt.Errorf("type '%s' not found in imported module '%s'", typeName, moduleName)
}

func deriveSemanticFunctionType(function *ast.FunctionType, module *Module) (*FunctionType, error) {
	var params []Type
	for _, param := range function.Parameters {
		paramType, err := DeriveSemanticType(param, module)
		if err != nil {
			return nil, err
		}
		params = append(params, paramType)
	}
	var returnType Type
	if function.ReturnType != nil {
		retType, err := DeriveSemanticType(function.ReturnType, module)
		if err != nil {
			return nil, err
		}
		returnType = retType
	}
	return &FunctionType{
		Parameters: params,
		ReturnType: returnType,
		Name:       function.TypeName,
	}, nil
}

func deriveSemanticStructFromAst(structType *ast.StructType, module *Module) (*StructType, error) {
	fields := make(map[string]Type)
	for _, field := range structType.Fields {
		if field.FieldType != nil {
			fieldName := field.FieldIdentifier.Name
			fieldType, err := DeriveSemanticType(field.FieldType, module)
			if err != nil {
				return nil, err
			}
			fields[fieldName] = fieldType
		}
	}
	return &StructType{
		Name:   structType.TypeName,
		Fields: fields,
	}, nil
}

func deriveSemanticArrayType(array *ast.ArrayType, module *Module) (Type, error) {
	elementType, err := DeriveSemanticType(array.ElementType, module)
	if err != nil {
		return nil, err
	}
	return &ArrayType{
		ElementType: elementType,
		Name:        array.TypeName,
	}, nil
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
