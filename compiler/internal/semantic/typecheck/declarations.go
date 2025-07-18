package typecheck

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"fmt"
)

func checkVariableDeclaration(r *analyzer.AnalyzerNode, varDecl *ast.VarDeclStmt, cm *ctx.Module) {
	for i, variable := range varDecl.Variables {

		variableInModule, _ := cm.SymbolTable.Lookup(variable.Identifier.Name)
		initializer := varDecl.Initializers[i]
		typeToAdd := inferTypeFromExpression(r, initializer, cm)

		if variable.ExplicitType == nil {
			colors.CYAN.Printf("Infering type for variable '%s'\n", variable.Identifier.Name)

			variableInModule.Type = typeToAdd

			if r.Debug {
				colors.CYAN.Printf("Inferred type for variable '%s': %s\n", variable.Identifier.Name, typeToAdd)
			}
		} else if variable.ExplicitType != nil && typeToAdd != nil {
			//both explicit type and initializer are provided. they must match
			explicitType := getDatatype(variable.ExplicitType)
			if IsAssignableFrom(explicitType, typeToAdd) {
				colors.CYAN.Printf("Variable '%s' type matches explicit type: %s\n", variable.Identifier.Name, typeToAdd)
			} else {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, initializer.Loc(), fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'", typeToAdd.String(), variable.Identifier.Name, explicitType.String()), report.TYPECHECK_PHASE)
				return
			}
		}
	}
}
