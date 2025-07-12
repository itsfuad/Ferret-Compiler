package typecheck

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/types"
)

// CheckProgram performs type checking on the entire program
func CheckProgram(r *analyzer.AnalyzerNode) {
	for _, node := range r.Program.Nodes {
		checkNode(r, node)
	}
	if r.Debug {
		colors.GREEN.Printf("Type checked '%s'\n", r.Program.FullPath)
	}
}

// checkNode performs type checking on a single AST node
func checkNode(r *analyzer.AnalyzerNode, node ast.Node) {
	switch n := node.(type) {
	case *ast.ImportStmt:
		checkImportStmt(r, n)
	case *ast.VarDeclStmt:
		checkVarDecl(r, n)
	case *ast.AssignmentStmt:
		checkAssignment(r, n)
	case *ast.ExpressionStmt:
		checkExpressionStmt(r, n)
	case *ast.TypeDeclStmt:
		checkTypeDecl(r, n)
	// Add more cases as needed
	default:
		// Skip nodes that don't need type checking
	}
}

// checkImportStmt performs type checking on import statements
func checkImportStmt(r *analyzer.AnalyzerNode, stmt *ast.ImportStmt) {
	//check the imported module
	importModule, err := r.Ctx.GetModule(stmt.ImportPath.Value)
	if err != nil {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			stmt.Loc(),
			err.Error(),
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	if importModule.AST == nil {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			stmt.Loc(),
			"imported module has no AST: "+stmt.ImportPath.Value,
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	//typecheck it
	anz := analyzer.NewAnalyzerNode(importModule.AST, r.Ctx, r.Debug)
	CheckProgram(anz)
}

// checkVarDecl performs type checking on variable declarations
func checkVarDecl(r *analyzer.AnalyzerNode, stmt *ast.VarDeclStmt) {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		return
	}

	for i, v := range stmt.Variables {
		sym, found := currentModule.SymbolTable.Lookup(v.Identifier.Name)
		if !found {
			continue // Error should have been reported by resolver
		}

		if i < len(stmt.Initializers) && stmt.Initializers[i] != nil {
			checkVariableInitializer(r, v, sym, stmt.Initializers[i])
		}
	}
}

// checkVariableInitializer checks the type compatibility of a variable initializer
func checkVariableInitializer(r *analyzer.AnalyzerNode, v *ast.VariableToDeclare, sym *semantic.Symbol, initializer ast.Expression) {
	initType := inferExpressionType(r, initializer)

	if sym.Type != nil {
		// Explicit type provided - check compatibility
		checkTypeCompatibility(r, v, sym, initType)
	} else {
		// No explicit type provided - perform type inference
		performTypeInference(r, v, sym, initType)
	}
}

// checkTypeCompatibility validates that an initializer type is compatible with the variable type
func checkTypeCompatibility(r *analyzer.AnalyzerNode, v *ast.VariableToDeclare, sym *semantic.Symbol, initType semantic.Type) {
	if initType != nil {
		// Resolve type aliases for both target and source types
		targetType := resolveTypeAlias(r, sym.Type)
		sourceType := resolveTypeAlias(r, initType)

		if !semantic.IsAssignableFrom(targetType, sourceType) {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				v.Identifier.Loc(),
				"type mismatch: cannot assign "+initType.String()+" to "+sym.Type.String(),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
		}
	}
}

// performTypeInference infers the type of a variable from its initializer
func performTypeInference(r *analyzer.AnalyzerNode, v *ast.VariableToDeclare, sym *semantic.Symbol, initType semantic.Type) {
	if initType != nil {
		// Update the symbol's type with the inferred type
		sym.Type = initType
	} else {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			v.Identifier.Loc(),
			"cannot infer type: initializer expression is invalid",
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
	}
}

// checkAssignment performs type checking on assignments
func checkAssignment(r *analyzer.AnalyzerNode, stmt *ast.AssignmentStmt) {
	// Check each assignment pair
	leftExprs := *stmt.Left
	rightExprs := *stmt.Right

	for i, leftExpr := range leftExprs {
		if i >= len(rightExprs) {
			break // Mismatched assignment count - should be caught elsewhere
		}

		leftType := inferExpressionType(r, leftExpr)
		rightType := inferExpressionType(r, rightExprs[i])

		if leftType != nil && rightType != nil {
			if !semantic.IsAssignableFrom(leftType, rightType) {
				r.Ctx.Reports.Add(
					r.Program.FullPath,
					leftExpr.Loc(),
					"type mismatch: cannot assign "+rightType.String()+" to "+leftType.String(),
					report.TYPECHECK_PHASE,
				).SetLevel(report.SEMANTIC_ERROR)
			}
		}
	}
}

// checkExpressionStmt performs type checking on expression statements
func checkExpressionStmt(r *analyzer.AnalyzerNode, stmt *ast.ExpressionStmt) {
	if stmt.Expressions != nil {
		for _, expr := range *stmt.Expressions {
			inferExpressionType(r, expr) // This will catch type errors in expressions
		}
	}
}

// checkTypeDecl performs type checking on type declarations
func checkTypeDecl(r *analyzer.AnalyzerNode, stmt *ast.TypeDeclStmt) {
	// Basic validation - more sophisticated checks can be added here
	if stmt.BaseType != nil {
		checkTypeValidity(r, stmt.BaseType)
	}
}

// checkTypeValidity checks if a type is valid (exists and is well-formed)
func checkTypeValidity(r *analyzer.AnalyzerNode, dataType ast.DataType) bool {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		return false
	}

	switch t := dataType.(type) {
	case *ast.UserDefinedType:
		// Check if the user-defined type exists
		_, found := currentModule.SymbolTable.Lookup(string(t.TypeName))
		if !found {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				t.Loc(),
				"undefined type: "+string(t.TypeName),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
			return false
		}
		return true
	case *ast.ArrayType:
		return checkTypeValidity(r, t.ElementType)
	case *ast.StructType:
		// Check all field types
		for _, field := range t.Fields {
			if field.FieldType != nil && !checkTypeValidity(r, field.FieldType) {
				return false
			}
		}
		return true
	default:
		// Primitive types are always valid
		return true
	}
}

// inferExpressionType infers the type of an expression
func inferExpressionType(r *analyzer.AnalyzerNode, expr ast.Expression) semantic.Type {
	if expr == nil {
		return nil
	}

	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		return nil
	}

	var resultType semantic.Type

	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		resultType = inferIdentifierType(currentModule, e)
	case *ast.StringLiteral:
		resultType = semantic.CreatePrimitiveType(types.STRING)
	case *ast.IntLiteral:
		resultType = semantic.CreatePrimitiveType(types.INT32)
	case *ast.FloatLiteral:
		resultType = semantic.CreatePrimitiveType(types.FLOAT64)
	case *ast.BoolLiteral:
		resultType = semantic.CreatePrimitiveType(types.BOOL)
	case *ast.ByteLiteral:
		resultType = semantic.CreatePrimitiveType(types.BYTE)
	case *ast.FieldAccessExpr:
		resultType = inferFieldAccessType(r, e)
	case *ast.BinaryExpr:
		resultType = inferBinaryExprType(r, e)
	case *ast.VarScopeResolution:
		resultType = inferVarScopeResolutionType(r, e)
	case *ast.StructLiteralExpr:
		resultType = inferStructLiteralType(r, currentModule, e)
	case *ast.ArrayLiteralExpr:
		resultType = inferArrayLiteralType(r, e)
	case *ast.IndexableExpr:
		resultType = inferIndexableType(r, e)
	case *ast.TypeScopeResolution:
		resultType = inferTypeScopeResolutionType(r, e)
	default:
		resultType = nil
	}

	logInferredType(r, expr, resultType)
	return resultType
}

// inferBinaryOperationType infers the result type of a binary operation
func inferBinaryOperationType(operator string, leftType, rightType semantic.Type) semantic.Type {
	switch operator {
	case "+", "-", "*", "/", "%":
		return inferArithmeticOperationType(operator, leftType, rightType)
	case "==", "!=", "<", "<=", ">", ">=":
		return inferComparisonOperationType(leftType, rightType)
	case "&&", "||":
		return inferLogicalOperationType(leftType, rightType)
	case "&", "|", "^", "<<", ">>":
		return inferBitwiseOperationType(leftType, rightType)
	default:
		return nil
	}
}

// inferArithmeticOperationType handles arithmetic operations
func inferArithmeticOperationType(operator string, leftType, rightType semantic.Type) semantic.Type {
	// Arithmetic operations - return common numeric type
	commonType := semantic.GetCommonType(leftType, rightType)
	if commonType != nil {
		return commonType
	}

	// String concatenation with + (only str + str)
	if operator == "+" {
		leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
		rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

		if leftOk && rightOk &&
			leftPrim.Name == types.STRING && rightPrim.Name == types.STRING {
			return semantic.CreatePrimitiveType(types.STRING)
		}
	}

	// If we reach here, the operation is invalid
	return nil
}

// inferComparisonOperationType handles comparison operations
func inferComparisonOperationType(leftType, rightType semantic.Type) semantic.Type {
	// Check if types are comparable
	if semantic.IsAssignableFrom(leftType, rightType) ||
		semantic.IsAssignableFrom(rightType, leftType) ||
		semantic.GetCommonType(leftType, rightType) != nil {
		return semantic.CreatePrimitiveType(types.BOOL)
	}
	return nil
}

// inferLogicalOperationType handles logical operations
func inferLogicalOperationType(leftType, rightType semantic.Type) semantic.Type {
	leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
	rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

	if leftOk && rightOk &&
		leftPrim.Name == types.BOOL && rightPrim.Name == types.BOOL {
		return semantic.CreatePrimitiveType(types.BOOL)
	}
	return nil
}

// inferBitwiseOperationType handles bitwise operations
func inferBitwiseOperationType(leftType, rightType semantic.Type) semantic.Type {
	leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
	rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

	if leftOk && rightOk && isIntegerType(leftPrim.Name) && isIntegerType(rightPrim.Name) {
		commonType := semantic.GetCommonType(leftType, rightType)
		if commonType != nil {
			return commonType
		}
	}
	return nil
}

// isIntegerType checks if a type is an integer type
func isIntegerType(typeName types.TYPE_NAME) bool {
	switch typeName {
	case types.INT8, types.INT16, types.INT32, types.INT64,
		types.UINT8, types.UINT16, types.UINT32, types.UINT64, types.BYTE:
		return true
	default:
		return false
	}
}

// resolveTypeAlias resolves a type alias to its underlying type
func resolveTypeAlias(r *analyzer.AnalyzerNode, t semantic.Type) semantic.Type {
	userType, ok := t.(*semantic.UserType)
	if !ok {
		return t
	}

	// Try to find the type in current module first
	if resolvedType := resolveTypeInCurrentModule(r, userType); resolvedType != nil {
		return resolvedType
	}

	// If not found locally, check imported modules
	if resolvedType := resolveTypeInImportedModules(r, userType); resolvedType != nil {
		return resolvedType
	}

	// If not found, return the original type
	return t
}

// resolveTypeInCurrentModule tries to resolve a type in the current module
func resolveTypeInCurrentModule(r *analyzer.AnalyzerNode, userType *semantic.UserType) semantic.Type {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		return nil
	}

	sym, found := currentModule.SymbolTable.Lookup(string(userType.Name))
	if found && sym.Kind == semantic.SymbolType && sym.Type != nil {
		return sym.Type
	}
	return nil
}

// resolveTypeInImportedModules tries to resolve a type in imported modules
func resolveTypeInImportedModules(r *analyzer.AnalyzerNode, userType *semantic.UserType) semantic.Type {
	typeName := string(userType.Name)

	for _, moduleName := range r.Ctx.ModuleNames() {
		module, err := r.Ctx.GetModule(moduleName)
		if err != nil {
			continue
		}

		sym, found := module.SymbolTable.Lookup(typeName)
		if found && sym.Kind == semantic.SymbolType && sym.Type != nil {
			return sym.Type
		}
	}
	return nil
}

// Helper functions for inferExpressionType to reduce cognitive complexity

// inferIdentifierType infers the type of an identifier expression
func inferIdentifierType(currentModule *ctx.Module, e *ast.IdentifierExpr) semantic.Type {
	sym, found := currentModule.SymbolTable.Lookup(e.Name)
	if found {
		return sym.Type
	}
	return nil
}

// inferFieldAccessType infers the type of a field access expression
func inferFieldAccessType(r *analyzer.AnalyzerNode, e *ast.FieldAccessExpr) semantic.Type {
	objectType := inferExpressionType(r, *e.Object)
	if structType, ok := objectType.(*semantic.StructType); ok {
		fieldType := structType.GetFieldType(e.Field.Name)
		if fieldType == nil {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				e.Field.Loc(),
				"field '"+e.Field.Name+"' not found in "+structType.String(),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
		}
		return fieldType
	}
	return nil
}

// inferBinaryExprType infers the type of a binary expression
func inferBinaryExprType(r *analyzer.AnalyzerNode, e *ast.BinaryExpr) semantic.Type {
	leftType := inferExpressionType(r, *e.Left)
	rightType := inferExpressionType(r, *e.Right)

	if leftType == nil || rightType == nil {
		return nil
	}

	resultType := inferBinaryOperationType(e.Operator.Value, leftType, rightType)
	if resultType == nil {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"invalid binary operation: "+leftType.String()+" "+e.Operator.Value+" "+rightType.String(),
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
	}
	return resultType
}

// inferVarScopeResolutionType infers the type of a variable scope resolution expression
func inferVarScopeResolutionType(r *analyzer.AnalyzerNode, e *ast.VarScopeResolution) semantic.Type {
	moduleName := e.Module.Name
	varName := e.Var.Name

	importModuleName, ok := r.Program.ModulenameToImportpath[moduleName]
	if !ok {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"module not found: "+moduleName,
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}

	importedModule, err := r.Ctx.GetModule(importModuleName)
	if err != nil {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			err.Error(),
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}

	sym, found := importedModule.SymbolTable.Lookup(varName)
	if !found {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"variable '"+varName+"' not found in module '"+moduleName+"'",
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}
	return sym.Type
}

// inferStructLiteralType infers the type of a struct literal expression
func inferStructLiteralType(r *analyzer.AnalyzerNode, currentModule *ctx.Module, e *ast.StructLiteralExpr) semantic.Type {
	if e.IsAnonymous {
		panic("Anonymous struct literals are not yet supported")
		return nil
	}

	if e.StructName == nil {
		return nil
	}

	structTypeName := e.StructName.Name
	sym, found := currentModule.SymbolTable.Lookup(structTypeName)
	if !found {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"struct type '"+structTypeName+"' not found",
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}
	// TODO: Validate that the fields match the struct definition
	return sym.Type
}

// inferArrayLiteralType infers the type of an array literal expression
func inferArrayLiteralType(r *analyzer.AnalyzerNode, e *ast.ArrayLiteralExpr) semantic.Type {
	if len(e.Elements) == 0 {
		// Empty array - cannot infer type
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"cannot infer array type from empty array literal",
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}

	// Infer type from first element
	firstElementType := inferExpressionType(r, e.Elements[0])
	if firstElementType == nil {
		return nil
	}

	// Check that all elements have compatible types
	commonType := firstElementType
	for _, element := range e.Elements {
		elementType := inferExpressionType(r, element)
		if elementType == nil {
			continue
		}

		// Try to find a common type
		newCommonType := semantic.GetCommonType(commonType, elementType)
		if newCommonType == nil {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				element.Loc(),
				"array element type mismatch: cannot use "+elementType.String()+" in array of "+commonType.String(),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
			return nil
		}
		commonType = newCommonType
	}

	// Create array type with the common element type
	return semantic.CreateArrayType(commonType)
}

// inferIndexableType infers the type of an array/map indexing expression
func inferIndexableType(r *analyzer.AnalyzerNode, e *ast.IndexableExpr) semantic.Type {
	// Get the type of the indexable expression
	indexableType := inferExpressionType(r, *e.Indexable)
	if indexableType == nil {
		return nil
	}

	// Check if it's an array type
	if arrayType, ok := indexableType.(*semantic.ArrayType); ok {
		// Verify the index is an integer type
		indexType := inferExpressionType(r, *e.Index)
		if indexType == nil {
			return nil
		}

		// Check if index type is an integer
		if !isIntegerTypeForIndexing(indexType) {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				(*e.Index).Loc(),
				"array index must be an integer type, got "+indexType.String(),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
			return nil
		}

		// Return the element type of the array
		return arrayType.ElementType
	}

	// If not an array, report error
	r.Ctx.Reports.Add(
		r.Program.FullPath,
		(*e.Indexable).Loc(),
		"cannot index non-array type "+indexableType.String(),
		report.TYPECHECK_PHASE,
	).SetLevel(report.SEMANTIC_ERROR)
	return nil
}

// isIntegerTypeForIndexing checks if a type can be used as an array index
func isIntegerTypeForIndexing(t semantic.Type) bool {
	if primType, ok := t.(*semantic.PrimitiveType); ok {
		return isIntegerType(primType.Name)
	}
	return false
}

// inferTypeScopeResolutionType infers the type of a type scope resolution expression
func inferTypeScopeResolutionType(r *analyzer.AnalyzerNode, e *ast.TypeScopeResolution) semantic.Type {
	moduleName := e.Module.Name
	typeName := string(e.Type())

	importModuleName, ok := r.Program.ModulenameToImportpath[moduleName]
	if !ok {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"module not found: "+moduleName,
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}

	importedModule, err := r.Ctx.GetModule(importModuleName)
	if err != nil {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			err.Error(),
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}

	sym, found := importedModule.SymbolTable.Lookup(typeName)
	if !found {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			e.Loc(),
			"type '"+typeName+"' not found in module '"+moduleName+"'",
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}
	// For TypeScopeResolution, we return the actual type, not the symbol's type
	return sym.Type
}

// logInferredType logs the inferred type for debugging
func logInferredType(r *analyzer.AnalyzerNode, expr ast.Expression, resultType semantic.Type) {
	if r.Debug {
		if resultType == nil {
			colors.YELLOW.Printf("Inferred type for expression '%v': <nil>\n", expr)
		} else {
			colors.YELLOW.Printf("Inferred type for expression '%v': %s\n", expr, resultType.String())
		}
	}
}
