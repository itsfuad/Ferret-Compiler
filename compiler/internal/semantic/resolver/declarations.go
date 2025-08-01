package resolver

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/stype"
	"compiler/internal/symbol"
	"compiler/internal/types"
	"fmt"
)

func resolveFunctionDecl(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) {
	functionSymbol, found := cm.SymbolTable.Lookup(fn.Identifier.Name)
	if !found {
		colors.RED.Printf("Function '%s' not found in symbol table\n", fn.Identifier.Name)
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function '"+fn.Identifier.Name+"' is not declared", report.RESOLVER_PHASE)
		return
	}

	// Resolve parameter types and update function scope symbols
	paramTypes := resolveParameterTypes(r, fn, cm)
	if paramTypes == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to resolve parameter types for function '"+fn.Identifier.Name+"'", report.RESOLVER_PHASE)
		return // Error occurred during parameter resolution
	}

	// Resolve return type
	returnType := resolveReturnType(r, fn, cm)
	if returnType == nil {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Failed to resolve return type for function '"+fn.Identifier.Name+"'", report.RESOLVER_PHASE)
		return // Error occurred during return type resolution
	}

	fmt.Printf("Resolved function '%s' with parameters: %v and return type: %v\n", fn.Identifier.Name, paramTypes, returnType)
	// Resolve function body
	resolveFunctionBody(r, fn, cm)

	// Create and assign function type
	functionType := stype.FunctionType{
		Parameters: paramTypes,
		ReturnType: returnType,
	}
	functionSymbol.Type = &functionType
}

func resolveParameterTypes(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) []stype.Type {
	paramTypes := make([]stype.Type, 0) // Initialize as empty slice, not nil slice

	// Check if function has no parameters
	if fn.Function == nil || fn.Function.Params == nil || len(fn.Function.Params) == 0 {
		return paramTypes // Return empty slice for functions with no parameters
	}

	functionScope, exists := cm.FunctionScopes[fn.Identifier.Name]
	if !exists {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function scope for '"+fn.Identifier.Name+"' not found", report.RESOLVER_PHASE)
		return nil // Return nil to indicate error
	}

	for _, param := range fn.Function.Params {
		paramType, err := semantic.DeriveSemanticType(param.Type, cm)
		if err != nil {
			colors.RED.Printf("Error deriving type for parameter '%s': %s\n", param.Identifier.Name, err.Error())
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, param.Type.Loc(), "Invalid parameter type: "+err.Error(), report.RESOLVER_PHASE)
			return nil // Return nil to indicate error
		}
		paramTypes = append(paramTypes, paramType)

		// Update parameter symbol in function scope
		updateParameterSymbol(&param, paramType, functionScope, exists)
	}
	return paramTypes
}

func updateParameterSymbol(param *ast.Parameter, paramType stype.Type, functionScope *symbol.SymbolTable, exists bool) {
	if exists && param.Identifier != nil {
		if paramSymbol, found := functionScope.Lookup(param.Identifier.Name); found {
			paramSymbol.Type = paramType
		}
	}
}

func resolveReturnType(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) stype.Type {
	if fn.Function != nil && fn.Function.ReturnType != nil {
		retType, err := semantic.DeriveSemanticType(fn.Function.ReturnType, cm)
		if err != nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Function.ReturnType.Loc(), "Invalid return type: "+err.Error(), report.RESOLVER_PHASE)
			return nil
		}
		return retType
	}
	return &stype.PrimitiveType{Name: types.VOID}
}

func resolveFunctionBody(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *modules.Module) {
	// Check if function has a body
	if fn.Function == nil || fn.Function.Body == nil {
		return
	}

	colors.PINK.Printf("Resolving function body for '%s' at %s\n", fn.Identifier.Name, fn.Function.Body.Loc().String())

	functionScope, exists := cm.FunctionScopes[fn.Identifier.Name]
	if exists {
		colors.LIGHT_GREEN.Printf("Resolving function body for '%s' at %s\n", fn.Identifier.Name, fn.Function.Body.Loc().String())
		// Temporarily switch to function scope for body resolution
		// but preserve imports from module scope
		originalTable := cm.SymbolTable
		functionScope.Imports = originalTable.Imports // Copy imports to function scope
		cm.SymbolTable = functionScope
		resolveNode(r, fn.Function.Body, cm)
		cm.SymbolTable = originalTable // Restore module scope
	} else {
		colors.RED.Printf("Function scope for '%s' not found, resolving in module scope\n", fn.Identifier.Name)
		// Fallback to module scope if function scope not found
		resolveNode(r, fn.Function.Body, cm)
	}
}

func resolveVariableDeclaration(r *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *modules.Module) {

	colors.ORANGE.Printf("Resolving variable declaration\n")

	for i, variable := range decl.Variables {

		colors.BLUE.Printf("Resolving variable declaration '%s' at %s\n", variable.Identifier.Name, variable.Identifier.Loc().String())

		// Check initializer expression if present
		if i < len(decl.Initializers) && decl.Initializers[i] != nil {
			resolveExpr(r, decl.Initializers[i], cm)
		}

		// Look up the already-declared symbol from the collector phase
		symbol, found := cm.SymbolTable.Lookup(variable.Identifier.Name)
		if !found {
			colors.RED.Printf("Variable '%s' not found in symbol table\n", variable.Identifier.Name)
			r.Ctx.Reports.AddCriticalError(r.Program.FullPath, variable.Identifier.Loc(), "Variable '"+variable.Identifier.Name+"' was not collected during symbol collection phase", report.RESOLVER_PHASE)
			continue
		}

		if variable.ExplicitType != nil {
			got, err := semantic.DeriveSemanticType(variable.ExplicitType, cm)
			if err != nil {
				colors.RED.Printf("Error deriving type for variable '%s': %s\n", variable.Identifier.Name, err.Error())
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.ExplicitType.Loc(), "Invalid explicit type for variable declaration: "+err.Error(), report.RESOLVER_PHASE)
				continue
			}
			// Update the symbol's type
			symbol.Type = got
			if r.Debug {
				colors.TEAL.Printf("Declared variable symbol '%s' with explicit type '%v' at %s\n", variable.Identifier.Name, symbol.Type, variable.Identifier.Loc().String())
			}
		}
	}
}

func resolveTypeDeclaration(r *analyzer.AnalyzerNode, decl *ast.TypeDeclStmt, cm *modules.Module) {
	aliasName := decl.Alias.Name
	if aliasName == "" {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.Alias.Loc(), "Type alias name cannot be empty", report.RESOLVER_PHASE)
		return
	}

	// Look up the already-declared symbol from the collector phase
	symbol, found := cm.SymbolTable.Lookup(aliasName)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, decl.Alias.Loc(), "Type alias '"+aliasName+"' was not collected during symbol collection phase", report.RESOLVER_PHASE)
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

	// Update the symbol's type
	symbol.Type = symbolType

	if r.Debug {
		colors.ORANGE.Printf("Resolved type alias '%v', Def: %v at %s\n", symbol.Type, symbol.Type.(*stype.UserType).Definition, decl.Alias.Loc().String())
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
