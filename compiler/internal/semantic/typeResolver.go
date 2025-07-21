package semantic

import (
	"ferret/compiler/internal/ctx"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/semantic/stype"
	atype "ferret/compiler/internal/types"
	"fmt"
)

// DeriveSemanticType converts an AST DataType to a semantic stype.Type
func DeriveSemanticType(astType ast.DataType, module *ctx.Module) (stype.Type, error) {

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
	case *ast.UserDefinedType:
		return resolveUserDefinedType(t, module)
	default:
		return nil, fmt.Errorf("unsupported AST type: %T", t)
	}
}

func derivePrimitiveTypeFromAst(astType ast.DataType) (*stype.PrimitiveType, error) {
	return &stype.PrimitiveType{
		Name: astType.Type(),
	}, nil
}

func resolveUserDefinedType(userType *ast.UserDefinedType, module *ctx.Module) (stype.Type, error) {
	symbol, found := module.SymbolTable.Lookup(string(userType.TypeName))
	if !found {
		return nil, fmt.Errorf("user-defined type '%s' not found", userType.TypeName)
	}
	if symbol.Type == nil {
		return nil, fmt.Errorf("user-defined type '%s' has no associated type", userType.TypeName)
	}
	return symbol.Type, nil
}

func resolveTypeInImportedModule(res *ast.TypeScopeResolution, module *ctx.Module) (stype.Type, error) {
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
		return &stype.UserType{Name: symbol.Type.TypeName(), Definition: nil}, nil // No definition available
	}
	return nil, fmt.Errorf("type '%s' not found in imported module '%s'", typeName, moduleName)
}

func deriveSemanticFunctionType(function *ast.FunctionType, module *ctx.Module) (*stype.FunctionType, error) {
	var params []stype.Type
	for _, param := range function.Parameters {
		paramType, err := DeriveSemanticType(param, module)
		if err != nil {
			return nil, err
		}
		params = append(params, paramType)
	}
	var returnType stype.Type
	if function.ReturnType != nil {
		retType, err := DeriveSemanticType(function.ReturnType, module)
		if err != nil {
			return nil, err
		}
		returnType = retType
	}
	return &stype.FunctionType{
		Parameters: params,
		ReturnType: returnType,
		Name:       function.TypeName,
	}, nil
}

func deriveSemanticStructFromAst(structType *ast.StructType, module *ctx.Module) (*stype.StructType, error) {
	fields := make(map[string]stype.Type)
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
	return &stype.StructType{
		Name:   structType.TypeName,
		Fields: fields,
	}, nil
}

func deriveSemanticArrayType(array *ast.ArrayType, module *ctx.Module) (stype.Type, error) {
	elementType, err := DeriveSemanticType(array.ElementType, module)
	if err != nil {
		return nil, err
	}
	return &stype.ArrayType{
		ElementType: elementType,
		Name:        array.TypeName,
	}, nil
}

// IsStringType checks if a type is string
func IsStringType(t stype.Type) bool {
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return prim.Name == atype.STRING
	}
	return false
}

// IsBoolType checks if a type is boolean
func IsBoolType(t stype.Type) bool {
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return prim.Name == atype.BOOL
	}
	return false
}

// IsNumericType checks if a type is numeric
func IsNumericType(t stype.Type) bool {
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return atype.IsNumericTypeName(prim.Name)
	}
	return false
}

// IsIntegerType checks if a type is an integer type
func IsIntegerType(t stype.Type) bool {
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return atype.IsIntegerTypeName(prim.Name)
	}
	return false
}
