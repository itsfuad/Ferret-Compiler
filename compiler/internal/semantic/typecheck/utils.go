package typecheck

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

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
