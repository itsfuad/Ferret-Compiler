package resolver

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/types"
)

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

	// Resolve the base type first
	resolveType(r, stmt.BaseType)

	// Convert AST type to semantic type
	semanticType := semantic.ASTToSemanticType(stmt.BaseType)
	sym := semantic.NewSymbolWithLocation(typeName, semantic.SymbolType, semanticType, stmt.Alias.Loc())
	currentModule.SymbolTable.Declare(typeName, sym)
}
