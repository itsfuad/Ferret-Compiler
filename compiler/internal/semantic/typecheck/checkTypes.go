package typecheck

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)

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
