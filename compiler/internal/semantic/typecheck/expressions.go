package typecheck

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic"
	"ferret/compiler/internal/semantic/analyzer"
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/source"
	"ferret/compiler/internal/types"
	"ferret/compiler/internal/utils"
	"fmt"
)

// evaluateExpressionType infers the semantic type from an AST expression
func evaluateExpressionType(r *analyzer.AnalyzerNode, expr ast.Expression, cm *modules.Module) stype.Type {

	if expr == nil {
		return nil
	}

	var resultType stype.Type

	switch e := expr.(type) {
	// Literals
	case *ast.StringLiteral:
		resultType = &stype.PrimitiveType{TypeName: types.STRING}
	case *ast.IntLiteral:
		resultType = &stype.PrimitiveType{TypeName: types.INT32}
	case *ast.FloatLiteral:
		resultType = &stype.PrimitiveType{TypeName: types.FLOAT64}
	case *ast.BoolLiteral:
		resultType = &stype.PrimitiveType{TypeName: types.BOOL}
	case *ast.ByteLiteral:
		resultType = &stype.PrimitiveType{TypeName: types.BYTE}

	// Complex expressions
	case *ast.IdentifierExpr:
		resultType = checkIdentifierType(e, cm)
	case *ast.BinaryExpr:
		resultType = checkBinaryExprType(r, e, cm)
	case *ast.UnaryExpr:
		resultType = checkUnaryExprType(r, e, cm)
	case *ast.PrefixExpr:
		resultType = checkPrefixExprType(r, e, cm)
	case *ast.PostfixExpr:
		resultType = checkPostfixExprType(r, e, cm)
	case *ast.ArrayLiteralExpr:
		resultType = checkArrayLiteralType(r, e, cm)
	case *ast.IndexableExpr:
		resultType = checkIndexableType(r, e, cm)
	case *ast.VarScopeResolution:
		resultType = checkImportedSymbolType(r, e, cm)
	case *ast.FunctionCallExpr:
		resultType = checkFunctionCallType(r, e, cm)
	case *ast.FunctionLiteral:
		resultType = checkFunctionLiteralType(r, e, cm)
	case *ast.CastExpr:
		resultType = checkCastExprType(r, e, cm)
	case *ast.FieldAccessExpr:
		resultType = checkFieldAccessType(r, e, cm)
	case *ast.StructLiteralExpr:
		resultType = checkStructLiteralType(r, e, cm)
	default:
		// Unknown expression type
		resultType = &stype.Invalid{}
		r.Ctx.Reports.AddCriticalError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("Unsupported expression type <%T> for type inference", e),
			report.TYPECHECK_PHASE,
		)
	}

	return resultType
}

func checkFunctionCallType(r *analyzer.AnalyzerNode, call *ast.FunctionCallExpr, cm *modules.Module) stype.Type {
	// Get the type of the function being called
	functionType := evaluateExpressionType(r, *call.Caller, cm)
	if functionType == nil {
		return nil
	}

	// Verify it's a function type
	funcType, ok := functionType.(*stype.FunctionType)
	if !ok {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			call.Loc(),
			fmt.Sprintf("cannot call non-function type: %s", functionType),
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Check argument count
	expectedCount := len(funcType.Parameters)
	actualCount := len(call.Arguments)

	if expectedCount != actualCount {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			call.Loc(),
			fmt.Sprintf("function expects %d arguments, but %d were provided", expectedCount, actualCount),
			report.TYPECHECK_PHASE,
		)
		return funcType.ReturnType // Return the expected return type even with wrong arg count
	}

	// Check argument types
	for i, arg := range call.Arguments {
		argType := evaluateExpressionType(r, arg, cm)
		if argType == nil {
			continue // Skip if we can't determine argument type
		}

		expectedType := funcType.Parameters[i]
		if ok, err := isImplicitCastable(expectedType, argType); !ok {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				call.Loc(),
				fmt.Sprintf("%s argument error: %s",
					utils.NumericToOrdinal(i+1), err.Error()),
				report.TYPECHECK_PHASE,
			)
		}
	}

	// Return the function's return type (single return type now)
	return funcType.ReturnType
}

// checkFieldAccessType handles struct field access and method access
func checkFieldAccessType(r *analyzer.AnalyzerNode, fieldAccess *ast.FieldAccessExpr, cm *modules.Module) stype.Type {

	colors.PINK.Printf("Checking field access on field '%s' at %s\n", fieldAccess.Field.Name, fieldAccess.Loc())

	if fieldAccess.Object == nil || fieldAccess.Field == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			fieldAccess.Loc(),
			"Invalid field access expression",
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Evaluate the object being accessed
	objectType := evaluateExpressionType(r, *fieldAccess.Object, cm)
	if objectType == nil {
		return nil // Error already reported
	}

	fieldName := fieldAccess.Field.Name

	// Handle struct field/method access
	return checkStructFieldOrMethodAccess(r, objectType, fieldName, fieldAccess.Loc(), cm)
}

// checkStructFieldOrMethodAccess checks field or method access on struct types
func checkStructFieldOrMethodAccess(r *analyzer.AnalyzerNode, objectType stype.Type, propName string, location *source.Location, cm *modules.Module) stype.Type {
	// only struct types have fields, also user-defined types can be alias to structs
	// only user-defined types can have methods

	unwrapped := semantic.UnwrapType(objectType)
	if structType, ok := unwrapped.(*stype.StructType); ok {
		// Check for struct field
		if fieldType := findStructField(structType, propName); fieldType != nil {
			return fieldType // Return the field type if found
		}
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			location,
			fmt.Sprintf("Struct has no field named '%s'", propName),
			report.TYPECHECK_PHASE,
		)
		return nil // Field not found
	}

	if interfaceType, ok := unwrapped.(*stype.InterfaceType); ok {
		// Check for interface method
		if methodType, found := interfaceType.Methods[propName]; found {
			return methodType // Return the method type if found
		}
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			location,
			fmt.Sprintf("Interface has no method named '%s'", propName),
			report.TYPECHECK_PHASE,
		)
		return nil // Method not found
	}

	// Check for user-defined type methods
	if userType, ok := objectType.(*stype.UserType); ok {
		// Look up the method in the user type's definition
		userSymbol, found := cm.SymbolTable.Lookup(userType.Name)
		if !found {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				location,
				fmt.Sprintf("Unknown user type '%s'", userType.Name),
				report.TYPECHECK_PHASE,
			)
			return nil // User type not found
		}

		prop, found := userSymbol.SelfScope.Lookup(propName)
		if !found {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				location,
				fmt.Sprintf("User type '%s' has no method or field named '%s'", userType.Name, propName),
				report.TYPECHECK_PHASE,
			)
			return nil // Method/field not found
		}

		return prop.Type // Return the method/field type
	}

	return nil
}

// findStructField looks for a field in the struct type definition
func findStructField(structType *stype.StructType, fieldName string) stype.Type {
	if fieldType, exists := structType.Fields[fieldName]; exists {
		return fieldType
	}
	return nil
}

// checkStructLiteralType handles struct literal expressions like Person{name: "Alice", age: 30} or struct{x: 10, y: 20}
func checkStructLiteralType(r *analyzer.AnalyzerNode, structLiteral *ast.StructLiteralExpr, cm *modules.Module) stype.Type {
	// Check if this is an anonymous struct or named struct
	if structLiteral.IsAnonymous || structLiteral.StructName == nil {
		return checkAnonymousStructLiteral(r, structLiteral, cm)
	} else {
		return checkNamedStructLiteral(r, structLiteral, cm)
	}
}

// checkAnonymousStructLiteral handles unnamed struct literals like @struct{x: 10, y: 20}
func checkAnonymousStructLiteral(r *analyzer.AnalyzerNode, structLiteral *ast.StructLiteralExpr, cm *modules.Module) *stype.StructType {
	// Build the field map for the anonymous struct
	fields := make(map[string]stype.Type)

	for _, field := range structLiteral.Fields {
		if field.FieldIdentifier == nil {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				structLiteral.Loc(),
				"Anonymous struct field must have a name",
				report.TYPECHECK_PHASE,
			)
			continue
		}

		fieldName := field.FieldIdentifier.Name

		// Get the type of the field value
		if field.FieldValue != nil {
			fieldType := evaluateExpressionType(r, *field.FieldValue, cm)
			if fieldType != nil {
				fields[fieldName] = fieldType
			}
		} else {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				&field.Location,
				"Anonymous struct field must have a value",
				report.TYPECHECK_PHASE,
			)
		}
	}

	// Create an anonymous struct type
	return &stype.StructType{
		Fields: fields,
	}
}

// checkNamedStructLiteral handles named struct literals like @Person{name: "Alice", age: 30}
func checkNamedStructLiteral(r *analyzer.AnalyzerNode, structLiteral *ast.StructLiteralExpr, cm *modules.Module) *stype.UserType {
	// Look up the struct type by name
	structTypeName := structLiteral.StructName.Name
	symbol, found := cm.SymbolTable.Lookup(structTypeName)
	if !found {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			structLiteral.Loc(),
			fmt.Sprintf("Unknown struct type '%s'", structTypeName),
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Get the struct type from the symbol, handling UserType wrappers
	//symbol type must be user type
	userType, ok := symbol.Type.(*stype.UserType)
	if !ok {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			structLiteral.Loc(),
			fmt.Sprintf("type '%s' is not a user defined type", structTypeName),
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	unwrapped := semantic.UnwrapType(userType.Definition)
	structType, ok := unwrapped.(*stype.StructType)
	if !ok {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			structLiteral.Loc(),
			fmt.Sprintf("'%s' is not a struct type", structTypeName),
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	validateNamedStructFields(r, structLiteral, structType, structTypeName, cm)

	return userType // Return the user type wrapper
}

// validateNamedStructFields validates the fields in a named struct literal
func validateNamedStructFields(r *analyzer.AnalyzerNode, structLiteral *ast.StructLiteralExpr, structType *stype.StructType, structTypeName string, cm *modules.Module) {
	// Validate that all provided fields exist and have correct types
	providedFields := make(map[string]bool)
	for _, field := range structLiteral.Fields {

		fieldName := field.FieldIdentifier.Name
		providedFields[fieldName] = true

		// Check if the field exists in the struct definition
		expectedFieldType, exists := structType.Fields[fieldName]
		if !exists {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, &field.Location,
				fmt.Sprintf("Struct '%s' has no field named '%s'", structTypeName, fieldName),
				report.TYPECHECK_PHASE,
			)
			continue
		}

		// Check the type of the field value
		if field.FieldValue != nil {
			actualFieldType := evaluateExpressionType(r, *field.FieldValue, cm)
			if actualFieldType != nil {
				if ok, err := isImplicitCastable(expectedFieldType, actualFieldType); !ok {
					r.Ctx.Reports.AddSemanticError(
						r.Program.FullPath,
						&field.Location,
						fmt.Sprintf("Field error: %s", err.Error()),
						report.TYPECHECK_PHASE,
					)
				}
			}
		}
	}

	// Check that all required fields are provided
	for fieldName := range structType.Fields {
		if !providedFields[fieldName] {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				structLiteral.Loc(),
				fmt.Sprintf("Missing required field '%s' in struct literal for '%s'", fieldName, structTypeName),
				report.TYPECHECK_PHASE,
			)
		}
	}
}
