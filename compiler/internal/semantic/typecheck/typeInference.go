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
	if !validateStructLiteralFields(r, e, sym.Type) {
		return nil
	}
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
