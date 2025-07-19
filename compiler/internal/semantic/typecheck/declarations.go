package typecheck

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"fmt"
)

func checkVariableDeclaration(r *analyzer.AnalyzerNode, varDecl *ast.VarDeclStmt, cm *ctx.Module) {
	for i, variable := range varDecl.Variables {

		variableInModule, _ := cm.SymbolTable.Lookup(variable.Identifier.Name)
		initializer := varDecl.Initializers[i]
		typeToAdd := evaluateExpressionType(r, initializer, cm)

		if variable.ExplicitType == nil {
			variableInModule.Type = typeToAdd
			if r.Debug {
				colors.CYAN.Printf("Inferred type for variable '%s': %s\n", variable.Identifier.Name, typeToAdd)
			}
		} else if variable.ExplicitType != nil && typeToAdd != nil {
			//both explicit type and initializer are provided. they must match
			explicitType, err := ctx.DeriveSemanticType(variable.ExplicitType, cm)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.ExplicitType.Loc(), "Invalid explicit type for variable declaration: "+err.Error(), report.TYPECHECK_PHASE)
				return
			}
			if !IsAssignableFrom(explicitType, typeToAdd) {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, initializer.Loc(), fmt.Sprintf("cannot assign value of type '%s' to variable '%s' of type '%s'", typeToAdd.String(), variable.Identifier.Name, explicitType.String()), report.TYPECHECK_PHASE)
				return
			}
		}
	}
}
