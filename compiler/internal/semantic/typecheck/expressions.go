package typecheck

import (
	"errors"
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic"
	"ferret/compiler/internal/semantic/analyzer"
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/source"
	"ferret/compiler/internal/types"
	"fmt"
	"strings"
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

// checkCastExprType validates type cast expressions and returns the target type
func checkCastExprType(r *analyzer.AnalyzerNode, cast *ast.CastExpr, cm *modules.Module) stype.Type {
	// Evaluate the source expression type
	sourceType := evaluateExpressionType(r, *cast.Value, cm)
	if sourceType == nil {
		return nil
	}

	// Convert AST target type to semantic type
	targetType, err := semantic.DeriveSemanticType(cast.TargetType, cm)
	if err != nil || targetType == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			cast.Loc(),
			fmt.Sprintf("invalid target type in cast expression: %v", err),
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Check if the cast is valid
	isValid, err := isExplicitCastable(sourceType, targetType)
	if err != nil || !isValid {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			cast.Loc(),
			fmt.Sprintf("cannot cast from '%s' to '%s': %v", sourceType.String(), targetType.String(), err),
			report.TYPECHECK_PHASE,
		)
		return nil // Return nil if cast is invalid
	}

	return targetType
}

func isExplicitCastable(sourceType, targetType stype.Type) (bool, error) {
	// primitive types can be casted to each other
	if sourceType == nil || targetType == nil {
		return false, errors.New("source or target type is nil")
	}
	
	if isImplicitCastable(sourceType, targetType) {
		return true, nil // Implicit cast is also valid for explicit cast
	}

	sourceUnwrapped := semantic.UnwrapType(sourceType) // Unwrap any type aliases
	targetUnwrapped := semantic.UnwrapType(targetType) // Unwrap any type aliases

	// Check if both are primitive types
	if _, ok := sourceUnwrapped.(*stype.PrimitiveType); ok {
		if _, ok := targetUnwrapped.(*stype.PrimitiveType); ok {
			// Both are primitive types, check if they are castable
			return isPrimitiveExplicitCastable(sourceUnwrapped, targetUnwrapped)
		}
	}

	//structs can be casted to other struct and interfaces and interfaces can be casted to structs
	if ss, ok := sourceUnwrapped.(*stype.StructType); ok {
		//target must be a struct or interface. for now skip interface
		if ts, ok := targetUnwrapped.(*stype.StructType); ok {
			//both are struct
			return isStructExplicitCastable(ss, ts)
		}
	}

	return false, fmt.Errorf("no valid cast found from '%s' to '%s'", sourceType.String(), targetType.String())
}

func isStructExplicitCastable(sourceType, targetType *stype.StructType) (bool, error) {
	//targets all properties must present

	fieldErrors := make([]string, 0, len(targetType.Fields))

	for fieldName, fieldType := range targetType.Fields {
		if sourceFieldType, exists := sourceType.Fields[fieldName]; !exists {
			fieldErrors = append(fieldErrors, colors.RED.Sprintf(" - missing field '%s'", fieldName))
		} else if !isImplicitCastable(sourceFieldType, fieldType) {
			fieldErrors = append(fieldErrors, colors.PURPLE.Sprintf(" - field '%s' type required '%s', but got '%s'", fieldName, fieldType.String(), sourceFieldType.String()))
		}
	}

	if len(fieldErrors) > 0 {
		errMsg := colors.WHITE.Sprintf("\n%s and %s has field missmatch\n", sourceType.String(), targetType.String())
		return false, fmt.Errorf("%s%s", errMsg, strings.Join(fieldErrors, "\n"))
	}

	fmt.Printf("Successfully casted struct '%s' to '%s'\n", sourceType.String(), targetType.String())

	return true, nil
}

func isPrimitiveExplicitCastable(sourceType, targetType stype.Type) (bool, error) {
	sourcePrim, sOk := sourceType.(*stype.PrimitiveType)
	targetPrim, tOk := targetType.(*stype.PrimitiveType)

	if !sOk || !tOk {
		return false, errors.New("both source and target must be primitive types")
	}

	// Allow ALL numeric to numeric casting with explicit "as" keyword
	// The developer explicitly requests the conversion, so allow both widening and narrowing
	if types.IsNumericTypeName(sourcePrim.Name) && types.IsNumericTypeName(targetPrim.Name) {
		return true, nil
	}

	// Special case: byte can be cast to/from u8 and i8
	if sourcePrim.Name == types.BYTE {
		res := targetPrim.Name == types.UINT8 || targetPrim.Name == types.INT8
		if res {
			return res, nil
		}
		return false, fmt.Errorf("byte and %s are incompatible types", targetPrim.Name)
	}
	if targetPrim.Name == types.BYTE {
		res := sourcePrim.Name == types.UINT8 || sourcePrim.Name == types.INT8
		if res {
			return res, nil
		}
		return false, fmt.Errorf("byte and %s are incompatible types", sourcePrim.Name)
	}

	// No valid cast found
	return false, fmt.Errorf("no valid cast found from '%s' to '%s'", sourceType.String(), targetType.String())
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
