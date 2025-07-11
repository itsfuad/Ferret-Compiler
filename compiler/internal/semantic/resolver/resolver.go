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
)

type Resolver struct {
	ctx     *ctx.CompilerContext
	program *ast.Program
	Debug   bool
}

func NewResolver(program *ast.Program, ctx *ctx.CompilerContext, debug bool) *Resolver {
	colors.BLUE.Printf("New Resolver set for: %s\n", program.FullPath)
	return &Resolver{
		ctx:     ctx,
		program: program,
		Debug:   debug,
	}
}

func (r *Resolver) ResolveProgram() {
	if r.Debug {
		fmt.Printf("[Resolver] Starting semantic analysis for %s\n", r.program.FullPath)
	}

	for _, node := range r.program.Nodes {
		fmt.Printf("[Resolver] Resolving node: %T\n", node)
		resolveNode(r, node)
	}
	if r.Debug {
		colors.GREEN.Printf("[Resolver] ----------- Finished semantic analysis for %s ------------\n", r.program.FullPath)
	}
}

func resolveNode(r *Resolver, node ast.Node) {
	colors.PURPLE.Printf("Resolving node: %s\n", r.program.ImportPath)
	currentModule := r.ctx.GetModule(r.program.ImportPath)
	if currentModule == nil {
		r.ctx.Reports.Add(r.program.FullPath, r.program.Loc(), "module not found for node: "+r.program.ImportPath+"\n"+r.program.FullPath, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
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
	case *ast.TypeScopeResolution:
		resolveTypeScopeResolution(r, n)
	// Basic data types - these are primitive types that don't need special resolution
	case *ast.StringType:
		// String type is a primitive, no additional resolution needed
	case *ast.IntType:
		// Integer type is a primitive, no additional resolution needed
	case *ast.FloatType:
		// Float type is a primitive, no additional resolution needed
	case *ast.BoolType:
		// Boolean type is a primitive, no additional resolution needed
	case *ast.ByteType:
		// Byte type is a primitive, no additional resolution needed
	default:
		fmt.Printf("[Resolver] Node <%T> is not implemented yet\n", n)
		os.Exit(-1)
	}
}

func resolveTypeDecl(r *Resolver, stmt *ast.TypeDeclStmt) {
	// check if type is already declared or built-in or keyword
	typeName := stmt.Alias.Name
	if lexer.IsKeyword(typeName) || types.IsPrimitiveType(typeName) {
		r.ctx.Reports.Add(r.program.FullPath, stmt.Alias.Loc(), "cannot declare type with reserved keyword: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}
	//declare the type in the current module
	currentModule := r.ctx.GetModule(r.program.ImportPath)
	if currentModule == nil {
		r.ctx.Reports.Add(r.program.FullPath, stmt.Alias.Loc(), "<type decl> current module not found for type declaration: "+r.program.ImportPath, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}
	currentModule.SymbolTable.Declare(typeName, &semantic.Symbol{Name: typeName, Kind: semantic.SymbolType, Type: stmt.BaseType})
}

func resolveImport(r *Resolver, currentModule *ctx.Module, importStmt *ast.ImportStmt) {
	if r.Debug {
		fmt.Printf("[Resolver] Resolving import: %s\n", importStmt.ImportPath.Value)
	}
	if importStmt.ModuleName != "" && importStmt.FullPath != "" {
		importModule := r.ctx.GetModule(importStmt.ImportPath.Value)
		if importModule == nil {
			r.ctx.Reports.Add(r.program.FullPath, importStmt.Loc(), "<import resolver> module not found: "+importStmt.ModuleName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
		colors.GREEN.Printf("Retrieved module key '%s' for import alias '%s'\n", importStmt.ImportPath.Value, importStmt.ModuleName)
		importModuleAST := importModule.AST
		//semantic.AddPreludeSymbols(importModule.SymbolTable)
		resolver := NewResolver(importModuleAST, r.ctx, r.Debug)
		resolver.ResolveProgram()
		currentModule.SymbolTable.Imports[importStmt.ModuleName] = importModule.SymbolTable
	}
}

func resolveVarDecl(r *Resolver, stmt *ast.VarDeclStmt) {
	currentModuleImportpath := r.program.ImportPath
	for i, v := range stmt.Variables {
		name := v.Identifier.Name
		kind := semantic.SymbolVar
		if stmt.IsConst {
			kind = semantic.SymbolConst
		}
		// Type checking: ensure explicit type exists if provided
		currentModule := r.ctx.GetModule(currentModuleImportpath)
		if currentModule == nil {
			r.ctx.Reports.Add(r.program.FullPath, v.Identifier.Loc(), "<var decl resolver> module not found: "+currentModuleImportpath, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
			return
		}

		if v.ExplicitType != nil {
			resolveNode(r, v.ExplicitType)
		}

		sym := &semantic.Symbol{Name: name, Kind: kind, Type: v.ExplicitType}

		err := currentModule.SymbolTable.Declare(name, sym)
		if err != nil {
			// Redeclaration error
			r.ctx.Reports.Add(r.program.FullPath, v.Identifier.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
		// Check initializer expression if present
		if i < len(stmt.Initializers) && stmt.Initializers[i] != nil {
			resolveExpr(r, stmt.Initializers[i])
		}
	}
}

func resolveAssignment(r *Resolver, stmt *ast.AssignmentStmt) { // Check that all left-hand side variables are declared
	for _, lhs := range *stmt.Left {
		if id, ok := lhs.(*ast.IdentifierExpr); ok {
			varSym, found := r.ctx.Modules[r.program.FullPath].SymbolTable.Lookup(id.Name)
			if !found {
				r.ctx.Reports.Add(r.program.FullPath, id.Loc(), "assignment to undeclared variable: "+id.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			} else if varSym.Type != nil {
				// Type checking: ensure type exists for variable
				typeName := string(varSym.Type.Type())
				typeSym, found := r.ctx.Modules[r.program.FullPath].SymbolTable.Lookup(typeName)
				if !found || typeSym.Kind != semantic.SymbolType {
					r.ctx.Reports.Add(r.program.FullPath, id.Loc(), "unknown type for variable: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
		if _, found := r.ctx.Modules[r.program.FullPath].SymbolTable.Lookup(e.Name); !found {
			r.ctx.Reports.Add(r.program.FullPath, e.Loc(), "undeclared variable: "+e.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
	case *ast.BinaryExpr:
		resolveExpr(r, *e.Left)
		resolveExpr(r, *e.Right)
	case *ast.UnaryExpr:
		resolveExpr(r, *e.Operand)
	case *ast.PrefixExpr:
		resolveExpr(r, *e.Operand)
	case *ast.PostfixExpr:
		resolveExpr(r, *e.Operand)
	case *ast.FunctionCallExpr:
		resolveExpr(r, *e.Caller)
		for _, arg := range e.Arguments {
			resolveExpr(r, arg)
		}
	case *ast.FieldAccessExpr:
		resolveExpr(r, *e.Object)
	case *ast.VarScopeResolution:
		resolveVarScopeResolution(r, *e)
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
				resolveExpr(r, *field.FieldValue)
			}
		}
	case *ast.IndexableExpr:
		// Resolve the indexable expression and the index
		resolveExpr(r, *e.Indexable)
		resolveExpr(r, *e.Index)
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

func resolveTypeScopeResolution(r *Resolver, expr *ast.TypeScopeResolution) {

	modulename := expr.Module.Name
	if r.Debug {
		fmt.Printf("[Resolver] Resolving type from module: %s\n", modulename)
	}

	importModuleName, ok := r.program.ModulenameToImportpath[modulename]
	if !ok {
		r.ctx.Reports.Add(r.program.FullPath, expr.Module.Loc(), fmt.Sprintf("module '%s' not found", modulename), report.RESOLVER_PHASE).AddHint("Check if the module is imported correctly").SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Get the imported module's symbol table
	importModule := r.ctx.GetModule(importModuleName)
	if importModule == nil {
		r.ctx.Reports.Add(r.program.FullPath, expr.Module.Loc(), "<type scope> imported module not found: "+importModuleName, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	// Extract the type name from the type node
	var typeName string
	if userType, ok := expr.TypeNode.(*ast.UserDefinedType); ok {
		typeName = string(userType.TypeName)
	} else {
		r.ctx.Reports.Add(r.program.FullPath, expr.TypeNode.Loc(), "invalid type in scope resolution", report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Look up the type symbol in the imported module's symbol table
	symbol, found := importModule.SymbolTable.Lookup(typeName)
	if !found {
		r.ctx.Reports.Add(r.program.FullPath, expr.TypeNode.Loc(), fmt.Sprintf("type '%s' not found in module '%s'", typeName, modulename), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Verify it's actually a type
	if symbol.Kind != semantic.SymbolType {
		r.ctx.Reports.Add(r.program.FullPath, expr.TypeNode.Loc(), fmt.Sprintf("expected type but found variable '%s' in module '%s'", typeName, modulename), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	if r.Debug {
		fmt.Printf("[Resolver] Successfully resolved type '%s' from module '%s'\n", typeName, modulename)
	}
}

func resolveVarScopeResolution(r *Resolver, expr ast.VarScopeResolution) {
	modulename := expr.Module.Name
	if r.Debug {
		fmt.Printf("[Resolver] Resolving variable from module alias: %s\n", modulename)
	}

	importModuleName, ok := r.program.ModulenameToImportpath[modulename]
	if !ok {
		r.ctx.Reports.Add(r.program.FullPath, expr.Module.Loc(), fmt.Sprintf("module '%s' not found", modulename), report.RESOLVER_PHASE).AddHint("Check if the module is imported correctly").SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Get the imported module's symbol table
	importModule := r.ctx.GetModule(importModuleName)
	if importModule == nil {
		r.ctx.Reports.Add(r.program.FullPath, expr.Module.Loc(), "<var scope> imported module not found: "+importModuleName, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	// Look up the variable symbol in the imported module's symbol table
	symbol, found := importModule.SymbolTable.Lookup(expr.Var.Name)
	if !found {
		r.ctx.Reports.Add(r.program.FullPath, expr.Var.Loc(), fmt.Sprintf("variable '%s' not found in module '%s'", expr.Var.Name, modulename), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Verify it's actually a variable (not a type)
	if symbol.Kind == semantic.SymbolType {
		r.ctx.Reports.Add(r.program.FullPath, expr.Var.Loc(), fmt.Sprintf("expected variable but found type '%s' in module '%s'", expr.Var.Name, modulename), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	if r.Debug {
		fmt.Printf("[Resolver] Successfully resolved variable '%s' from module '%s'\n", expr.Var.Name, modulename)
	}
}
