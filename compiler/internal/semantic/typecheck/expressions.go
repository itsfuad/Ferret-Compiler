package typecheck

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic"
	"ferret/compiler/internal/semantic/analyzer"
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/source"
	"ferret/compiler/internal/types"
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
		resultType = &stype.PrimitiveType{Name: types.STRING}
	case *ast.IntLiteral:
		resultType = &stype.PrimitiveType{Name: types.INT32}
	case *ast.FloatLiteral:
		resultType = &stype.PrimitiveType{Name: types.FLOAT64}
	case *ast.BoolLiteral:
		resultType = &stype.PrimitiveType{Name: types.BOOL}
	case *ast.ByteLiteral:
		resultType = &stype.PrimitiveType{Name: types.BYTE}

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
		resultType = nil
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
			"cannot call non-function type: "+functionType.String(),
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
		if !isImplicitCastable(expectedType, argType) {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				call.Loc(),
				fmt.Sprintf("argument %d: cannot use %s as %s", i+1, argType.String(), expectedType.String()),
				report.TYPECHECK_PHASE,
			)
		}
	}

	// Return the function's return type (single return type now)
	return funcType.ReturnType
}

// checkFieldAccessType handles struct field access and method access
func checkFieldAccessType(r *analyzer.AnalyzerNode, fieldAccess *ast.FieldAccessExpr, cm *modules.Module) stype.Type {
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
func checkStructFieldOrMethodAccess(r *analyzer.AnalyzerNode, objectType stype.Type, fieldName string, location *source.Location, cm *modules.Module) stype.Type {
	// First, try to resolve the underlying struct type
	unwrapped := semantic.UnwrapType(objectType)
	structType, ok := unwrapped.(*stype.StructType)
	if !ok {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			location,
			fmt.Sprintf("Cannot access field '%s' on non-struct type '%s'", fieldName, objectType.String()),
			report.TYPECHECK_PHASE,
		)
	}

	// Try to find the field in the struct definition first
	if fieldType := findStructField(structType, fieldName); fieldType != nil {
		return fieldType
	}

	// Only named structs (UserType) can have methods
	// Anonymous structs (direct StructType) cannot have methods
	if _, isUserType := objectType.(*stype.UserType); isUserType {
		// Try to find a method in the struct's scope
		if methodType, err := findStructMethod(objectType, fieldName, cm); err == nil {
			return methodType
		}
	}

	// Neither field nor method found
	if _, isUserType := objectType.(*stype.UserType); isUserType {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			location,
			fmt.Sprintf("Struct '%s' has no field or method named '%s'", structType.String(), fieldName),
			report.TYPECHECK_PHASE,
		)
	} else {
		// Anonymous struct - only mention fields since methods aren't possible
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			location,
			fmt.Sprintf("Anonymous struct has no field named '%s'", fieldName),
			report.TYPECHECK_PHASE,
		)
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

// findStructMethod looks for a method in the struct's symbol scope
func findStructMethod(objectType stype.Type, methodName string, cm *modules.Module) (stype.Type, error) {
	// Only UserType (named structs) can have methods
	userType, ok := objectType.(*stype.UserType)
	if !ok {
		return nil, fmt.Errorf("cannot have method '%s' on unnamed struct type '%s'", methodName, objectType.String())
	}

	// Get the type name for symbol lookup
	structTypeName := string(userType.Name)

	// Look up the struct type symbol in the module
	if structSymbol, found := cm.SymbolTable.Lookup(structTypeName); found {
		if structSymbol.Scope != nil {
			// Look for the method in the struct's scope
			if methodSymbol, found := structSymbol.Scope.Lookup(methodName); found {
				return methodSymbol.Type, nil
			}
		}
	}

	return nil, fmt.Errorf("method '%s' not found in struct '%s'", methodName, structTypeName)
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
func checkAnonymousStructLiteral(r *analyzer.AnalyzerNode, structLiteral *ast.StructLiteralExpr, cm *modules.Module) stype.Type {
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
		Name:   types.TYPE_NAME(""), // Anonymous structs have no name
		Fields: fields,
	}
}

// checkNamedStructLiteral handles named struct literals like @Person{name: "Alice", age: 30}
func checkNamedStructLiteral(r *analyzer.AnalyzerNode, structLiteral *ast.StructLiteralExpr, cm *modules.Module) stype.Type {
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
	unwrapped := semantic.UnwrapType(symbol.Type)
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

	return validateNamedStructFields(r, structLiteral, structType, structTypeName, symbol.Type, cm)
}

// validateNamedStructFields validates the fields in a named struct literal
func validateNamedStructFields(r *analyzer.AnalyzerNode, structLiteral *ast.StructLiteralExpr, structType *stype.StructType, structTypeName string, symbolType stype.Type, cm *modules.Module) stype.Type {
	// Validate that all provided fields exist and have correct types
	providedFields := make(map[string]bool)
	for _, field := range structLiteral.Fields {
		if field.FieldIdentifier == nil {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				structLiteral.Loc(),
				"Struct field must have a name",
				report.TYPECHECK_PHASE,
			)
			continue
		}

		fieldName := field.FieldIdentifier.Name
		providedFields[fieldName] = true

		// Check if the field exists in the struct definition
		expectedFieldType, exists := structType.Fields[fieldName]
		if !exists {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				&field.Location,
				fmt.Sprintf("Struct '%s' has no field named '%s'", structTypeName, fieldName),
				report.TYPECHECK_PHASE,
			)
			continue
		}

		// Check the type of the field value
		if field.FieldValue != nil {
			actualFieldType := evaluateExpressionType(r, *field.FieldValue, cm)
			if actualFieldType != nil && !isImplicitCastable(expectedFieldType, actualFieldType) {
				r.Ctx.Reports.AddSemanticError(
					r.Program.FullPath,
					&field.Location,
					fmt.Sprintf("Field '%s' expects type '%s' but got '%s'", fieldName, expectedFieldType.String(), actualFieldType.String()),
					report.TYPECHECK_PHASE,
				)
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

	return symbolType
}
