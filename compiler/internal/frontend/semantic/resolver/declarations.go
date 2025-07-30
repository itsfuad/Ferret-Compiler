package resolver

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/semantic"
	"compiler/internal/frontend/semantic/analyzer"
	"compiler/internal/frontend/semantic/stype"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/symbol"
	"compiler/internal/types"
)

func resolveFunctionDecl(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) {

	symbol, found := cm.SymbolTable.Lookup(fn.Identifier.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function '"+fn.Identifier.Name+"' is not declared", report.RESOLVER_PHASE)
		return
	}

	//add the type information to the symbol
	var paramTypes []stype.Type
	if fn.Function.Params != nil {
		for _, param := range fn.Function.Params {
			paramType, err := semantic.DeriveSemanticType(param.Type, cm)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, param.Type.Loc(), "Invalid parameter type: "+err.Error(), report.RESOLVER_PHASE)
				return
			}
			paramTypes = append(paramTypes, paramType)
		}
	}

	var returnType stype.Type
	if fn.Function.ReturnType != nil {
		retType, err := semantic.DeriveSemanticType(fn.Function.ReturnType, cm)
		if err != nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Function.ReturnType.Loc(), "Invalid return type: "+err.Error(), report.RESOLVER_PHASE)
			return
		}
		returnType = retType
	} else {
		returnType = &stype.PrimitiveType{Name: types.VOID}
	}

	// Resolve function body
	if fn.Function.Body != nil {
		resolveNode(r, fn.Function.Body, cm)
	}

	// Create function type and symbol
	functionType := stype.FunctionType{
		Parameters: paramTypes,
		ReturnType: returnType,
	}

	symbol.Type = &functionType
}

func resolveVariableDeclaration(r *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *modules.Module) {
	for i, variable := range decl.Variables {

		var expType stype.Type

		// Check initializer expression if present
		if i < len(decl.Initializers) && decl.Initializers[i] != nil {
			resolveExpr(r, decl.Initializers[i], cm)
		}

		if variable.ExplicitType != nil {
			got, err := semantic.DeriveSemanticType(variable.ExplicitType, cm)
			if err != nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.ExplicitType.Loc(), "Invalid explicit type for variable declaration: "+err.Error(), report.RESOLVER_PHASE)
				return
			}
			expType = got
		}

		err := cm.SymbolTable.Declare(variable.Identifier.Name, symbol.NewSymbolWithLocation(variable.Identifier.Name, symbol.SymbolVar, expType, variable.Identifier.Loc()))
		if err != nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.Identifier.Loc(), "Failed to declare variable symbol: "+err.Error(), report.RESOLVER_PHASE)
			return
		}

		if r.Debug {
			colors.TEAL.Printf("Declared variable symbol '%s' with type '%v' at %s\n", variable.Identifier.Name, expType, variable.Identifier.Loc().String())
		}
	}
}

func resolveTypeDeclaration(r *analyzer.AnalyzerNode, decl *ast.TypeDeclStmt, cm *modules.Module) {
	aliasName := decl.Alias.Name
	if aliasName == "" {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.Alias.Loc(), "Type alias name cannot be empty", report.RESOLVER_PHASE)
		return
	}

	typeToDeclare, err := semantic.DeriveSemanticType(decl.BaseType, cm)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.BaseType.Loc(), "Invalid base type for type declaration: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	symbolType := &stype.UserType{
		Name:       types.TYPE_NAME(aliasName),
		Definition: typeToDeclare,
	}
	symbol := symbol.NewSymbolWithLocation(aliasName, symbol.SymbolType, symbolType, decl.Alias.Loc())

	err = cm.SymbolTable.Declare(aliasName, symbol)
	if err != nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.Alias.Loc(), "Failed to declare type alias: "+err.Error(), report.RESOLVER_PHASE)
		return
	}
	if r.Debug {
		colors.ORANGE.Printf("Declared type alias '%v', Def: %v at %s\n", symbol.Type, symbol.Type.(*stype.UserType).Definition, decl.Alias.Loc().String())
	}
}

func resolveAssignmentStmt(r *analyzer.AnalyzerNode, assign *ast.AssignmentStmt, cm *modules.Module) {
	// Resolve left-hand side expressions (assignees)
	if assign.Left != nil {
		resolveExpressionList(r, assign.Left, cm)
	}
	// Resolve right-hand side expressions (values)
	if assign.Right != nil {
		resolveExpressionList(r, assign.Right, cm)
	}

	if r.Debug {
		colors.TEAL.Printf("Resolved assignment statement at %s\n", assign.Loc().String())
	}
}
