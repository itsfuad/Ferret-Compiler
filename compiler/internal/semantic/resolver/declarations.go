package resolver

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/types"
)

// resolveTypeDecl handles type declaration resolution
func resolveTypeDecl(r *analyzer.AnalyzerNode, stmt *ast.TypeDeclStmt) {
	// check if type is already declared or built-in or keyword
	typeName := stmt.Alias.Name
	if lexer.IsKeyword(typeName) || types.IsPrimitiveType(typeName) {
		r.Ctx.Reports.Add(r.Program.FullPath, stmt.Alias.Loc(), "cannot declare type with reserved keyword: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}
	//declare the type in the current module
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, stmt.Alias.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	// Convert AST type to semantic type
	semanticType := semantic.ASTToSemanticType(stmt.BaseType)
	sym := semantic.NewSymbolWithLocation(typeName, semantic.SymbolType, semanticType, stmt.Alias.Loc())
	currentModule.SymbolTable.Declare(typeName, sym)
}

// resolveVarDecl handles variable declaration resolution
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
			resolveNode(r, v.ExplicitType)
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

// resolveAssignment handles assignment statement resolution
func resolveAssignment(r *analyzer.AnalyzerNode, stmt *ast.AssignmentStmt) {
	// Check that all left-hand side variables are declared
	for _, lhs := range *stmt.Left {
		if id, ok := lhs.(*ast.IdentifierExpr); ok {
			varSym, found := r.Ctx.Modules[r.Program.FullPath].SymbolTable.Lookup(id.Name)
			if !found {
				r.Ctx.Reports.Add(r.Program.FullPath, id.Loc(), "assignment to undeclared variable: "+id.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			} else if varSym.Type != nil {
				// Type checking: ensure type exists for variable
				typeName := string(varSym.Type.TypeName())
				typeSym, found := r.Ctx.Modules[r.Program.FullPath].SymbolTable.Lookup(typeName)
				if !found || typeSym.Kind != semantic.SymbolType {
					r.Ctx.Reports.Add(r.Program.FullPath, id.Loc(), "unknown type for variable: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
				}
			}
		} else {
			resolveExpr(r, lhs)
		}
	}
	// Check right-hand side expressions
	for _, rhs := range *stmt.Right {
		resolveExpr(r, rhs)
	}
}
