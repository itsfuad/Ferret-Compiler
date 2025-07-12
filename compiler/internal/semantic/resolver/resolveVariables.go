package resolver

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

func resolveVarDecl(r *analyzer.AnalyzerNode, stmt *ast.VarDeclStmt) {
	currentModuleImportpath := r.Program.ImportPath
	for i, v := range stmt.Variables {
		name := v.Identifier.Name
		kind := semantic.SymbolVar
		if stmt.IsConst {
			kind = semantic.SymbolConst
		}
		// Type checking: ensure explicit type exists if provided
		currentModule, err := r.Ctx.GetModule(currentModuleImportpath)
		if err != nil {
			r.Ctx.Reports.Add(r.Program.FullPath, v.Identifier.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
			return
		}

		if v.ExplicitType != nil {
			resolveType(r, v.ExplicitType)
		}

		// Convert AST type to semantic type
		var semanticType semantic.Type
		if v.ExplicitType != nil {
			semanticType = semantic.ASTToSemanticType(v.ExplicitType)
		}

		sym := semantic.NewSymbolWithLocation(name, kind, semanticType, v.Identifier.Loc())

		err = currentModule.SymbolTable.Declare(name, sym)
		if err != nil {
			// Redeclaration error
			r.Ctx.Reports.Add(r.Program.FullPath, v.Identifier.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
		// Check initializer expression if present
		if i < len(stmt.Initializers) && stmt.Initializers[i] != nil {
			resolveExpr(r, stmt.Initializers[i])
		}
	}
}
