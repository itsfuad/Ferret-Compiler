package resolver

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

func resolveFunctionDecl(r *analyzer.AnalyzerNode, stmt *ast.FunctionDecl) {
	functionName := stmt.Identifier.Name
	colors.YELLOW.Printf("[Resolver] Resolving function declaration: %s\n", functionName)

	if lexer.IsKeyword(functionName) {
		r.Ctx.Reports.Add(r.Program.FullPath, stmt.Identifier.Loc(), "cannot declare function with reserved keyword: "+functionName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, stmt.Identifier.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	// Check if function is already declared
	if _, exists := currentModule.SymbolTable.Lookup(functionName); exists {
		r.Ctx.Reports.Add(r.Program.FullPath, stmt.Identifier.Loc(), "function already declared: "+functionName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Create function type from AST
	var paramTypes []semantic.Type
	if stmt.Function.Params != nil {
		for _, param := range stmt.Function.Params {
			if param.Type == nil {
				r.Ctx.Reports.Add(r.Program.FullPath, &param.Identifier.Location, "parameter type must be specified", report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
				return
			}
			resolveASTNode(r, param.Type)
			paramType := semantic.ASTToSemanticType(param.Type)
			paramTypes = append(paramTypes, paramType)
		}
	}

	var returnTypes []semantic.Type
	if stmt.Function.ReturnType != nil {
		for _, ret := range stmt.Function.ReturnType {
			resolveASTNode(r, ret)
			retType := semantic.ASTToSemanticType(ret)
			returnTypes = append(returnTypes, retType)
		}
	}

	// Resolve function body
	if stmt.Function.Body != nil {
		resolveASTNode(r, stmt.Function.Body)
	}

	// Create function type and symbol
	functionType := semantic.CreateFunctionType(paramTypes, returnTypes)
	functionSymbol := semantic.NewSymbolWithLocation(functionName, semantic.SymbolFunc, functionType, stmt.Identifier.Loc())

	// Declare in symbol table
	currentModule.SymbolTable.Declare(functionName, functionSymbol)
}
