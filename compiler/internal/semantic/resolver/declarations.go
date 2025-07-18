package resolver

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"fmt"
)

func resolveFunctionDecl(r *analyzer.AnalyzerNode, fn *ast.FunctionDecl, cm *ctx.Module) {
	//identifier empty check is done in collector phase, so we can assume it's not empty here

	//now add the function definition to the current module's symbol table
	currentModule, _ := r.Ctx.GetModule(r.Program.ImportPath) // no error since already checked in collector phase
	symbol, found := currentModule.SymbolTable.Lookup(fn.Identifier.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, fn.Loc(), "Function '"+fn.Identifier.Name+"' is not declared", report.RESOLVER_PHASE)
		return
	}
	//add the type information to the symbol
	var paramTypes []semantic.Type
	if fn.Function.Params != nil {
		for _, param := range fn.Function.Params {
			if param.Type == nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, &param.Identifier.Location, "parameter type must be specified", report.RESOLVER_PHASE)
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
	for i, variable := range decl.Variables {

		var expType semantic.Type

		// Check initializer expression if present
		if i < len(decl.Initializers) && decl.Initializers[i] != nil {
			go resolveExpr(r, decl.Initializers[i], cm)
		}

		if variable.ExplicitType != nil {
			expType = semantic.ASTToSemanticType(variable.ExplicitType)
			if expType == nil {
				r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.ExplicitType.Loc(), "Invalid explicit type for variable declaration", report.RESOLVER_PHASE)
				return
			}
		}

		err := cm.SymbolTable.Declare(variable.Identifier.Name, semantic.NewSymbolWithLocation(variable.Identifier.Name, semantic.SymbolVar, expType, variable.Identifier.Loc()))
		if err != nil {
			r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.Identifier.Loc(), "Failed to declare variable symbol: "+err.Error(), report.RESOLVER_PHASE)
			return
		}

		if r.Debug {
			colors.TEAL.Printf("Declared variable symbol '%s' with type '%v' at %s\n", variable.Identifier.Name, expType, variable.Identifier.Loc().String())
		}
	}
}

func resolveExpr(r *analyzer.AnalyzerNode, expr ast.Expression, cm *ctx.Module) {
	if expr == nil {
		panic("resolveExpr called with nil expression")
	}
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		resolveIdentifier(r, e, cm)
	case *ast.BinaryExpr:
		go resolveExpr(r, *e.Left, cm)
		go resolveExpr(r, *e.Right, cm)
	case *ast.UnaryExpr:
		resolveExpr(r, *e.Operand, cm)
	case *ast.PrefixExpr:
		resolveExpr(r, *e.Operand, cm)
	case *ast.PostfixExpr:
		resolveExpr(r, *e.Operand, cm)
	case *ast.FunctionCallExpr:
		//add later
	case *ast.FieldAccessExpr:
		resolveExpr(r, *e.Object, cm)
	case *ast.VarScopeResolution:
		resolveImportedSymbol(r, e, cm)

	// Literal expressions - no resolution needed, just validate they exist
	case *ast.StringLiteral:
		// String literals don't need resolution
	case *ast.IntLiteral:
		// Integer literals don't need resolution
	case *ast.FloatLiteral:
		// Float literals don't need resolution
	case *ast.BoolLiteral:
		// Boolean literals don't need resolution
	case *ast.ByteLiteral:
		// Byte literals don't need resolution
	case *ast.ArrayLiteralExpr:
		//add later
	case *ast.StructLiteralExpr:
		//add later
	case *ast.IndexableExpr:
		resolveExpr(r, *e.Indexable, cm)
		resolveExpr(r, *e.Index, cm)
	case *ast.FunctionLiteral:
		//add later
	default:
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, expr.Loc(), fmt.Sprintf("Expression <%T> is not implemented yet", e), report.RESOLVER_PHASE)
	}
}

func resolveIdentifier(r *analyzer.AnalyzerNode, id *ast.IdentifierExpr, cm *ctx.Module) {
	if _, found := cm.SymbolTable.Lookup(id.Name); !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, id.Loc(), "undefined symbol: "+id.Name, report.RESOLVER_PHASE)
	}
}

func resolveImportedSymbol(r *analyzer.AnalyzerNode, res *ast.VarScopeResolution, cm *ctx.Module) {
	//find if the module exists
	moduleKey, found := r.Program.ModulenameToImportpath[res.Module.Name]
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Module '%s' is not imported", res.Module.Name), report.RESOLVER_PHASE)
		return
	}

	module, err := r.Ctx.GetModule(moduleKey)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, res.Loc(), "Failed to get imported module: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	if _, found := module.SymbolTable.Lookup(res.Identifier.Name); !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, res.Loc(), fmt.Sprintf("Symbol '%s' not found in module '%s'", res.Identifier.Name, moduleKey), report.RESOLVER_PHASE)
		return
	}

	if r.Debug {
		//print symbol X found in module Y imported from Z
		colors.TEAL.Printf("Resolved imported symbol '%s' from module '%s' imported from '%s'\n", res.Identifier.Name, res.Module.Name, moduleKey)
	}
}
