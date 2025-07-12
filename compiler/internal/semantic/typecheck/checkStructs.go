package typecheck

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

// validateStructLiteralFields validates that all fields in a struct literal match the struct definition
func validateStructLiteralFields(r *analyzer.AnalyzerNode, e *ast.StructLiteralExpr, structType semantic.Type) bool {
	semanticStructType, ok := structType.(*semantic.StructType)
	if !ok {
		return reportStructTypeError(r, e, structType)
	}

	providedFields := make(map[string]bool)

	// Validate each field in the struct literal
	if !validateStructFields(r, e, semanticStructType, providedFields) {
		return false
	}

	// Check if all required fields are provided
	return validateRequiredFields(r, e, semanticStructType, providedFields)
}

// reportStructTypeError reports an error when the expected type is not a struct
func reportStructTypeError(r *analyzer.AnalyzerNode, e *ast.StructLiteralExpr, structType semantic.Type) bool {
	r.Ctx.Reports.Add(
		r.Program.FullPath,
		e.Loc(),
		"expected struct type, got "+structType.String(),
		report.TYPECHECK_PHASE,
	).SetLevel(report.SEMANTIC_ERROR)
	return false
}

// validateStructFields validates each field in the struct literal
func validateStructFields(r *analyzer.AnalyzerNode, e *ast.StructLiteralExpr, semanticStructType *semantic.StructType, providedFields map[string]bool) bool {
	for _, field := range e.Fields {
		if !validateSingleStructField(r, field, semanticStructType, providedFields) {
			return false
		}
	}
	return true
}

// validateSingleStructField validates a single field in the struct literal
func validateSingleStructField(r *analyzer.AnalyzerNode, field ast.StructField, semanticStructType *semantic.StructType, providedFields map[string]bool) bool {
	fieldName := field.FieldIdentifier.Name

	// Check if this field exists in the struct definition
	if !semanticStructType.HasField(fieldName) {
		r.Ctx.Reports.Add(
			r.Program.FullPath,
			field.FieldIdentifier.Loc(),
			"field '"+fieldName+"' does not exist in struct type '"+string(semanticStructType.Name)+"'",
			report.TYPECHECK_PHASE,
		).SetLevel(report.SEMANTIC_ERROR)
		return false
	}

	providedFields[fieldName] = true

	// Validate the field value type if present
	if field.FieldValue != nil {
		return validateFieldValueType(r, field, semanticStructType, fieldName)
	}
	return true
}

// validateFieldValueType validates the type of a field value
func validateFieldValueType(r *analyzer.AnalyzerNode, field ast.StructField, semanticStructType *semantic.StructType, fieldName string) bool {
	expectedFieldType := semanticStructType.GetFieldType(fieldName)
	actualFieldType := inferExpressionType(r, *field.FieldValue)

	if actualFieldType != nil && expectedFieldType != nil {
		// Resolve type aliases for both types
		expectedType := resolveTypeAlias(r, expectedFieldType)
		actualType := resolveTypeAlias(r, actualFieldType)

		if !semantic.IsAssignableFrom(expectedType, actualType) {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				(*field.FieldValue).Loc(),
				"type mismatch for field '"+fieldName+"': cannot assign "+actualType.String()+" to "+expectedType.String(),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
			return false
		}
	}
	return true
}

// validateRequiredFields checks if all required fields are provided in the struct literal
func validateRequiredFields(r *analyzer.AnalyzerNode, e *ast.StructLiteralExpr, semanticStructType *semantic.StructType, providedFields map[string]bool) bool {
	for fieldName := range semanticStructType.Fields {
		if !providedFields[fieldName] {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				e.Loc(),
				"missing field '"+fieldName+"' in struct literal",
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
			return false
		}
	}
	return true
}
