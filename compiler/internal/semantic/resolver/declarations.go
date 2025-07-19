package resolver

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/types"
)

func resolveFunctionDecl(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *ctx.Module) {

	symbol, found := cm.SymbolTable.Lookup(fn.Identifier.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function '"+fn.Identifier.Name+"' is not declared", report.RESOLVER_PHASE)
		return
	}

	//add the type information to the symbol
	var paramTypes []ctx.Type
	if fn.Function.Params != nil {
		for _, param := range fn.Function.Params {
			resolveNode(r, param.Type, cm)
			paramType, err := ctx.DeriveSemanticType(param.Type, cm)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, param.Type.Loc(), "Invalid parameter type: "+err.Error(), report.RESOLVER_PHASE)
				return
			}
			paramTypes = append(paramTypes, paramType)
		}
	}

	var returnTypes []ctx.Type
	if fn.Function.ReturnType != nil {
		for _, ret := range fn.Function.ReturnType {
			resolveNode(r, ret, cm)
			retType, err := ctx.DeriveSemanticType(ret, cm)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, ret.Loc(), "Invalid return type: "+err.Error(), report.RESOLVER_PHASE)
				return
			}
			returnTypes = append(returnTypes, retType)
		}
	}

	// Resolve function body
	if fn.Function.Body != nil {
		resolveNode(r, fn.Function.Body, cm)
	}

	// Create function type and symbol
	functionType := ctx.FunctionType{
		Parameters:  paramTypes,
		ReturnTypes: returnTypes,
	}

	symbol.Type = &functionType
}

func resolveVariableDeclaration(r *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *ctx.Module) {
	for i, variable := range decl.Variables {

		var expType ctx.Type

		// Check initializer expression if present
		if i < len(decl.Initializers) && decl.Initializers[i] != nil {
			resolveExpr(r, decl.Initializers[i], cm)
		}

		if variable.ExplicitType != nil {
			got, err := ctx.DeriveSemanticType(variable.ExplicitType, cm)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.ExplicitType.Loc(), "Invalid explicit type for variable declaration: "+err.Error(), report.RESOLVER_PHASE)
				return
			}
			expType = got
		}

		err := cm.SymbolTable.Declare(variable.Identifier.Name, ctx.NewSymbolWithLocation(variable.Identifier.Name, ctx.SymbolVar, expType, variable.Identifier.Loc()))
		if err != nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.Identifier.Loc(), "Failed to declare variable symbol: "+err.Error(), report.RESOLVER_PHASE)
			return
		}

		if r.Debug {
			colors.TEAL.Printf("Declared variable symbol '%s' with type '%v' at %s\n", variable.Identifier.Name, expType, variable.Identifier.Loc().String())
		}
	}
}

func resolveTypeDeclaration(r *analyzer.AnalyzerNode, decl *ast.TypeDeclStmt, cm *ctx.Module) {
	aliasName := decl.Alias.Name
	if aliasName == "" {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.Alias.Loc(), "Type alias name cannot be empty", report.RESOLVER_PHASE)
		return
	}

	typeToDeclare, err := ctx.DeriveSemanticType(decl.BaseType, cm)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.BaseType.Loc(), "Invalid base type for type declaration: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	symbolType := &ctx.UserType{
		Name:       types.TYPE_NAME(aliasName),
		Definition: typeToDeclare,
	}
	symbol := ctx.NewSymbolWithLocation(aliasName, ctx.SymbolType, symbolType, decl.Alias.Loc())

	err = cm.SymbolTable.Declare(aliasName, symbol)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.Alias.Loc(), "Failed to declare type alias: "+err.Error(), report.RESOLVER_PHASE)
		return
	}
	if r.Debug {
		colors.ORANGE.Printf("Declared type alias '%v', Def: %v at %s\n", symbol.Type, symbol.Type.(*ctx.UserType).Definition, decl.Alias.Loc().String())
	}
}
