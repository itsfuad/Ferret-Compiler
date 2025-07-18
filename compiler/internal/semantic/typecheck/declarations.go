package typecheck

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)

func checkVariableDeclaration(r *analyzer.AnalyzerNode, varDecl *ast.VarDeclStmt, cm *ctx.Module) {
	for i, variable := range varDecl.Variables {

		variableInModule, _ := cm.SymbolTable.Lookup(variable.Identifier.Name)
		initializer := varDecl.Initializers[i]
		typeToAdd := inferExpressionType(r, initializer, cm)

		if variable.ExplicitType == nil {
			colors.CYAN.Printf("Infering type for variable '%s'\n", variable.Identifier.Name)
			
			variableInModule.Type = typeToAdd

			if r.Debug {
				colors.CYAN.Printf("Inferred type for variable '%s': %s\n", variable.Identifier.Name, typeToAdd)
			}
		} else if variable.ExplicitType != nil && typeToAdd != nil {
			//both explicit type and initializer are provided. they must match
			if variableInModule.Type.Equals(typeToAdd) {
				colors.CYAN.Printf("Variable '%s' type matches explicit type: %s\n", variable.Identifier.Name, typeToAdd)
			} else {
				r.Ctx.Reports.Add(r.Program.FullPath, variable.ExplicitType.Loc(), "Explicit type does not match initializer type", report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
				return
			}
		}
	}
}