package resolver

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/types"
	"fmt"
	"os"
	"strings"
)

type Resolver struct {
	ctx     *ctx.CompilerContext
	program *ast.Program
	Debug   bool
}

func NewResolver(program *ast.Program, ctx *ctx.CompilerContext, debug bool) *Resolver {
	colors.BLUE.Printf("New Resolver set for: %s\n", program.FilePath)
	return &Resolver{
		ctx:     ctx,
		program: program,
		Debug:   debug,
	}
}

func (r *Resolver) ResolveProgram() {
	if r.Debug {
		fmt.Printf("[Resolver] Starting semantic analysis for %s\n", r.program.FilePath)
	}

	for _, node := range r.program.Nodes {
		fmt.Printf("[Resolver] Resolving node: %T\n", node)
		resolveNode(r, node)
	}
	if r.Debug {
		fmt.Printf("[Resolver] Finished semantic analysis for %s\n", r.program.FilePath)
	}
}

func resolveNode(r *Resolver, node ast.Node) {
	currentModuleName := r.ctx.AbsToModuleName(r.program.FilePath)
	currentModule := r.ctx.GetModule(currentModuleName)
	if currentModule == nil {
		r.ctx.Reports.Add(r.program.FilePath, r.program.Loc(), "current module not found: "+currentModuleName, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}
	switch n := node.(type) {
	case *ast.ImportStmt:
		resolveImport(r, currentModule, n)
	case *ast.VarDeclStmt:
		resolveVarDecl(r, n)
	case *ast.AssignmentStmt:
		resolveAssignment(r, n)
	case *ast.ExpressionStmt:
		resolveExpressionStmt(r, n)
	case *ast.TypeDeclStmt:
		resolveTypeDecl(r, n)
	default:
		fmt.Printf("[Resolver] Node <%T> is not implemented yet\n", n)
		os.Exit(-1)
	}
}

func resolveTypeDecl(r *Resolver, stmt *ast.TypeDeclStmt) {
	// check if type is already declared or built-in or keyword
	typeName := stmt.Alias.Name
	if lexer.IsKeyword(typeName) || types.IsPrimitiveType(typeName) {
		r.ctx.Reports.Add(r.program.FilePath, stmt.Alias.Loc(), "cannot declare type with reserved keyword: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}
	//declare the type in the current module
	moduleName := r.ctx.AbsToModuleName(r.program.FilePath)
	currentModule := r.ctx.GetModule(moduleName)
	if currentModule == nil {
		r.ctx.Reports.Add(r.program.FilePath, stmt.Alias.Loc(), "current module not found: "+moduleName, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}
	currentModule.SymbolTable.Declare(typeName, &semantic.Symbol{Name: typeName, Kind: semantic.SymbolType, Type: stmt.BaseType})
}

func resolveImport(r *Resolver, currentModule *ctx.Module, importStmt *ast.ImportStmt) {
	if r.Debug {
		fmt.Printf("[Resolver] Resolving import: %s\n", importStmt.ModuleName)
	}
	if importStmt.ModuleName != "" && importStmt.FilePath != "" {
		importModule := r.ctx.GetModule(importStmt.ImportPath.Value)
		if importModule == nil {
			r.ctx.Reports.Add(r.program.FilePath, importStmt.Loc(), "module not found: "+importStmt.ModuleName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
		colors.GREEN.Printf("Retrieved module '%s' for import alias '%s'\n", importStmt.ImportPath.Value, importStmt.ModuleName)
		importModuleAST := importModule.AST
		//semantic.AddPreludeSymbols(importModule.SymbolTable)
		resolver := NewResolver(importModuleAST, r.ctx, r.Debug)
		resolver.ResolveProgram()
		currentModule.SymbolTable.Imports[importStmt.ModuleName] = importModule.SymbolTable
	}
}

func resolveVarDecl(r *Resolver, stmt *ast.VarDeclStmt) {
	currentModuleName := r.ctx.AbsToModuleName(r.program.FilePath)
	for i, v := range stmt.Variables {
		name := v.Identifier.Name
		kind := semantic.SymbolVar
		if stmt.IsConst {
			kind = semantic.SymbolConst
		}
		// Type checking: ensure explicit type exists if provided
		currentModule := r.ctx.GetModule(currentModuleName)
		if currentModule == nil {
			r.ctx.Reports.Add(r.program.FilePath, v.Identifier.Loc(), "module not found: "+currentModuleName, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
			return
		}

		if v.ExplicitType != nil {
			typeName := string(v.ExplicitType.Type())

			var sym *semantic.Symbol
			var found bool

			if strings.Contains(typeName, "::") {
				parts := strings.Split(typeName, "::")
				moduleName := parts[0]
				symbolName := parts[1]
				module := r.ctx.GetModule(moduleName)
				if module == nil {
					r.ctx.Reports.Add(r.program.FilePath, v.Identifier.Loc(), "module not found: "+moduleName, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
					return
				}

				sym, found = module.SymbolTable.Lookup(symbolName)

				v.ExplicitType = sym.Type
			} else {
				sym, found = currentModule.SymbolTable.Lookup(typeName)
			}
			if !found || sym.Kind != semantic.SymbolType {
				r.ctx.Reports.Add(r.program.FilePath, v.Identifier.Loc(), "unknown type: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			}
		}

		sym := &semantic.Symbol{Name: name, Kind: kind, Type: v.ExplicitType}

		err := currentModule.SymbolTable.Declare(name, sym)
		if err != nil {
			// Redeclaration error
			r.ctx.Reports.Add(r.program.FilePath, v.Identifier.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
		// Check initializer expression if present
		if i < len(stmt.Initializers) && stmt.Initializers[i] != nil {
			resolveExpr(r, stmt.Initializers[i])
		}
	}
}

func resolveAssignment(r *Resolver, stmt *ast.AssignmentStmt) { // Check that all left-hand side variables are declared
	for _, lhs := range stmt.Left {
		if id, ok := lhs.(*ast.IdentifierExpr); ok {
			varSym, found := r.ctx.Modules[r.program.FilePath].SymbolTable.Lookup(id.Name)
			if !found {
				r.ctx.Reports.Add(r.program.FilePath, id.Loc(), "assignment to undeclared variable: "+id.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			} else if varSym.Type != nil {
				// Type checking: ensure type exists for variable
				typeName := string(varSym.Type.Type())
				typeSym, found := r.ctx.Modules[r.program.FilePath].SymbolTable.Lookup(typeName)
				if !found || typeSym.Kind != semantic.SymbolType {
					r.ctx.Reports.Add(r.program.FilePath, id.Loc(), "unknown type for variable: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
				}
			}
		} else {
			resolveExpr(r, lhs)
		}
	}
	// Check right-hand side expressions
	for _, rhs := range stmt.Right {
		resolveExpr(r, rhs)
	}
}

func resolveExpressionStmt(r *Resolver, stmt *ast.ExpressionStmt) {
	if stmt.Expressions != nil {
		for _, expr := range *stmt.Expressions {
			resolveExpr(r, expr)
		}
	}
}

func resolveExpr(r *Resolver, expr ast.Expression) {
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		if _, found := r.ctx.Modules[r.program.FilePath].SymbolTable.Lookup(e.Name); !found {
			r.ctx.Reports.Add(r.program.FilePath, e.Loc(), "undeclared variable: "+e.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
	case *ast.BinaryExpr:
		resolveExpr(r, e.Left)
		resolveExpr(r, e.Right)
	case *ast.UnaryExpr:
		resolveExpr(r, e.Operand)
	case *ast.PrefixExpr:
		resolveExpr(r, e.Operand)
	case *ast.PostfixExpr:
		resolveExpr(r, e.Operand)
	case *ast.FunctionCallExpr:
		resolveExpr(r, e.Caller)
		for _, arg := range e.Arguments {
			resolveExpr(r, arg)
		}
	case *ast.FieldAccessExpr:
		resolveExpr(r, e.Object)
	case *ast.ScopeResolutionExpr:
		resolveScopeResolution(r, e)
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
		// Resolve array elements
		for _, element := range e.Elements {
			resolveExpr(r, element)
		}
	case *ast.StructLiteralExpr:
		// Resolve struct field values
		for _, field := range e.Fields {
			if field.FieldValue != nil {
				resolveExpr(r, field.FieldValue)
			}
		}
	case *ast.IndexableExpr:
		// Resolve the indexable expression and the index
		resolveExpr(r, e.Indexable)
		resolveExpr(r, e.Index)
	case *ast.FunctionLiteral:
		// Resolve function body
		if e.Body != nil {
			for _, node := range e.Body.Nodes {
				resolveNode(r, node)
			}
		}
	default:
		fmt.Printf("[Resolver] Expression <%T> is not implemented yet\n", e)
	}
}

func resolveScopeResolution(r *Resolver, expr *ast.ScopeResolutionExpr) {
	alias := expr.Module.Name
	if r.Debug {
		fmt.Printf("[Resolver] Resolving module alias: %s\n", alias)
	}

	importModuleName, ok := r.ctx.AliasToModuleName[alias]
	if !ok {
		r.ctx.Reports.Add(r.program.FilePath, expr.Module.Loc(), fmt.Sprintf("module '%s' not found", alias), report.RESOLVER_PHASE).AddHint("Check if the module is imported correctly").SetLevel(report.SEMANTIC_ERROR)
		return
	}

	if r.Debug {
		fmt.Printf("[Resolver] Found module '%s' for alias '%s'\n", importModuleName, alias)
	}

	currentModuleName := r.ctx.AbsToModuleName(r.program.FilePath)
	if r.Debug {
		fmt.Printf("[Resolver] Current module: %s\n", currentModuleName)
	}

	currentModule := r.ctx.GetModule(currentModuleName)
	if currentModule == nil {
		r.ctx.Reports.Add(r.program.FilePath, expr.Module.Loc(), "current module not found: "+currentModuleName, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	fmt.Printf("[Resolver] Resolving '%s::%s'\n", alias, expr.Identifier.Name)

	importModuleSymbolTable, ok := currentModule.SymbolTable.Imports[alias]
	if !ok {
		r.ctx.Reports.Add(r.program.FilePath, expr.Module.Loc(), fmt.Sprintf("module '%s' is not imported in current module '%s'", alias, currentModuleName), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}
	//currecurrentModule.SymbolTable.Imports[alias] = importModule.SymbolTable
	if _, found := importModuleSymbolTable.Lookup(expr.Identifier.Name); !found {
		r.ctx.Reports.Add(r.program.FilePath, expr.Identifier.Loc(), fmt.Sprintf("symbol '%s' not found in module '%s'", expr.Identifier.Name, alias), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		if r.Debug {
			fmt.Printf("[Resolver] Symbol '%s' not found in module '%s' (file: %s)\n", expr.Identifier.Name, alias, importModuleName)
		}
	} else if r.Debug {
		fmt.Printf("[Resolver] Resolved '%s::%s' (module: %s)\n", alias, expr.Identifier.Name, importModuleName)
	}
}
