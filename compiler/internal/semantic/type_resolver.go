package semantic

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/semantic/stype"
	"compiler/internal/types"
	"fmt"
)

// UnwrapType resolves user types to their underlying types A -> B -> C(not user type), return C
// Note: This function requires access to symbol tables to properly resolve type aliases.
// For now, it only checks the Definition field. For full resolution, use resolveTypeAlias instead.
func UnwrapType(t stype.Type) stype.Type {
	if userType, ok := t.(*stype.UserType); ok {
		if userType.Definition != nil {
			return UnwrapType(userType.Definition)
		}
	}
	return t
}

// IsStringType checks if a stype.Type is string
func IsStringType(t stype.Type) bool {
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return prim.TypeName == types.STRING
	}
	return false
}

// IsBoolType checks if a stype.Type is boolean
func IsBoolType(t stype.Type) bool {
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return prim.TypeName == types.BOOL
	}
	return false
}

// IsNumericType checks if a stype.Type is numeric
func IsNumericType(t stype.Type) bool {
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return IsNumericTypeName(prim.TypeName)
	}
	return false
}

// IsIntegerType checks if a stype.Type is an integer stype.Type
func IsIntegerType(t stype.Type) bool {
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return IsIntegerTypeName(prim.TypeName)
	}
	return false
}

// IsVoidType checks if a stype.Type is void
func IsVoidType(t stype.Type) bool {
	t = UnwrapType(t) // Unwrap any type aliases
	if prim, ok := t.(*stype.PrimitiveType); ok {
		return prim.TypeName == types.VOID
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

// DeriveSemanticType converts an AST DataType to a semantic stype.Type
func DeriveSemanticType(astType ast.DataType, module *modules.Module) (stype.Type, error) {

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
	case *ast.InterfaceType:
		return deriveSemanticInterfaceType(t, module)
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
		TypeName: astType.Type(),
	}, nil
}

func resolveUserDefinedType(userType *ast.UserDefinedType, module *modules.Module) (stype.Type, error) {
	symbol, found := module.SymbolTable.Lookup(string(userType.TypeName))
	if !found {
		return nil, fmt.Errorf("type %q does not exist", userType.TypeName)
	}
	if symbol.Type == nil {
		// Check if this is a forward reference (type declared later)
		if symbol.Location != nil {
			usagePos := userType.Loc().Start
			declarationPos := symbol.Location.Start

			// If type is used before it's declared, that's a forward reference error
			if usagePos.Line < declarationPos.Line ||
				(usagePos.Line == declarationPos.Line && usagePos.Column < declarationPos.Column) {
				return nil, fmt.Errorf("cannot use type %q before it is declared",
					userType.TypeName)
			}
		}
		return nil, fmt.Errorf("type %q has no associated type", userType.TypeName)
	}
	return symbol.Type, nil
}

func resolveTypeInImportedModule(res *ast.TypeScopeResolution, module *modules.Module) (stype.Type, error) {
	// Handle type scope resolution (e.g., module::TypeName)
	moduleName := res.Module.Name
	typeName := res.TypeNode.Type().String()
	symbolTable, err := module.SymbolTable.GetImportedModule(moduleName)
	if err != nil {
		return nil, err
	}

	// Look up the type in the imported module's symbol table
	if symbol, ok := symbolTable.Lookup(typeName); ok {

		if module.UsedImports == nil {
			module.UsedImports = make(map[string]bool)
		}

		module.UsedImports[res.Module.Name] = true

		if symbol.Type != nil {
			return symbol.Type, nil
		}

		return &stype.UserType{Name: symbol.Name, Definition: nil}, nil // No definition available
	}

	return nil, fmt.Errorf("type %q not found in imported module %q", typeName, moduleName)
}

func deriveSemanticFunctionType(function *ast.FunctionType, module *modules.Module) (*stype.FunctionType, error) {
	var params []stype.ParamsType
	for _, param := range function.Parameters {
		paramType, err := DeriveSemanticType(param.Type, module)
		if err != nil {
			return nil, err
		}
		params = append(params, stype.ParamsType{Name: param.Identifier.Name, Type: paramType})
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
	}, nil
}

func deriveSemanticStructFromAst(structType *ast.StructType, module *modules.Module) (*stype.StructType, error) {
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
		Fields: fields,
	}, nil
}

func deriveSemanticInterfaceType(interfaceType *ast.InterfaceType, module *modules.Module) (stype.Type, error) {
	methods := make(map[string]*stype.FunctionType)
	for _, method := range interfaceType.Methods {
		methodType, err := DeriveSemanticType(method.Method, module)
		if err != nil {
			return nil, err
		}
		if functionType, ok := methodType.(*stype.FunctionType); ok {
			methods[method.Name.Name] = functionType
		} else {
			return nil, fmt.Errorf("interface method %q is not a function type", method.Name.Name)
		}
	}
	return &stype.InterfaceType{
		Methods: methods,
	}, nil
}

func deriveSemanticArrayType(array *ast.ArrayType, module *modules.Module) (stype.Type, error) {
	elementType, err := DeriveSemanticType(array.ElementType, module)
	if err != nil {
		return nil, err
	}
	return &stype.ArrayType{
		ElementType: elementType,
	}, nil
}
