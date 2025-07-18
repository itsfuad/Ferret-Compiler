package resolver

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)

func resolveFunctionDecl(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *ctx.Module) {
	//identifier empty check is done in collector phase, so we can assume it's not empty here
	
	//now add the function definition to the current module's symbol table
	currentModule, _ := r.Ctx.GetModule(r.Program.ImportPath) // no error since already checked in collector phase
	symbol, found := currentModule.SymbolTable.Lookup(fn.Identifier.Name); if !found {
		r.Ctx.Reports.Add(r.Program.FullPath, fn.Loc(), "Function '"+fn.Identifier.Name+"' is not declared", report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}
	//add the type information to the symbol
	var paramTypes []semantic.Type
	if fn.Function.Params != nil {
		for _, param := range fn.Function.Params {
			if param.Type == nil {
				r.Ctx.Reports.Add(r.Program.FullPath, &param.Identifier.Location, "parameter type must be specified", report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
				return
			}
			resolveNode(r, param.Type, cm)
			paramType := semantic.ASTToSemanticType(param.Type)
			paramTypes = append(paramTypes, paramType)
		}
	}

	var returnTypes []semantic.Type
	if fn.Function.ReturnType != nil {
		for _, ret := range fn.Function.ReturnType {
			resolveNode(r, ret, cm)
			retType := semantic.ASTToSemanticType(ret)
			returnTypes = append(returnTypes, retType)
		}
	}

	// Resolve function body
	if fn.Function.Body != nil {
		resolveNode(r, fn.Function.Body, cm)
	}

	// Create function type and symbol
	functionType := semantic.FunctionType{
		Parameters:  paramTypes,
		ReturnTypes: returnTypes,
	}

	symbol.Type = &functionType
}

func resolveVariableDeclaration(r *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *ctx.Module) {
	for _, variable := range decl.Variables {

		var expType semantic.Type

		if variable.ExplicitType != nil {
			expType = semantic.ASTToSemanticType(variable.ExplicitType)
			if expType == nil {
				r.Ctx.Reports.Add(r.Program.FullPath, variable.ExplicitType.Loc(), "Invalid explicit type for variable declaration", report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
				return
			}
		}
		err := cm.SymbolTable.Declare(variable.Identifier.Name, semantic.NewSymbolWithLocation(variable.Identifier.Name, semantic.SymbolVar, expType, variable.Identifier.Loc()))
		if err != nil {
			r.Ctx.Reports.Add(r.Program.FullPath, variable.Identifier.Loc(), "Failed to declare variable symbol: "+err.Error(), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			return
		}

		if r.Debug {
			if expType == nil {
				colors.YELLOW.Printf("Declared variable symbol '%s' with no explicit type at %s\n", variable.Identifier.Name, variable.Identifier.Loc().String())
			} else {
				colors.TEAL.Printf("Declared variable symbol '%s' with type '%v' at %s\n", variable.Identifier.Name, expType, variable.Identifier.Loc().String())
			}
		}
	}
}