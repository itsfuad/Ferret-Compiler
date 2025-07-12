package typecheck

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

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
