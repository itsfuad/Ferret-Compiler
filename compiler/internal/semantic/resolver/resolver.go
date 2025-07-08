package resolver

import (
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"fmt"
)

type Resolver struct {
	Symbols      *semantic.SymbolTable
	ctx          *ctx.CompilerContext
	File         string
	Reports      *report.Reports
	Debug        bool
	ModuleTables map[string]*semantic.SymbolTable // module file path -> symbol table
	AliasToPath  map[string]string                // import alias -> file path
}

func NewResolver(ctx *ctx.CompilerContext, file string, reports *report.Reports, debug bool) *Resolver {
	return &Resolver{
		Symbols:      semantic.NewSymbolTable(nil),
		ctx:          ctx,
		File:         file,
		Reports:      reports,
		Debug:        debug,
		ModuleTables: make(map[string]*semantic.SymbolTable),
		AliasToPath:  make(map[string]string),
	}
}

func (r *Resolver) ResolveProgram(prog *ast.Program) {
	if r.Debug {
		fmt.Printf("[Resolver] Starting semantic analysis for %s\n", r.File)
	}
	// Build import alias map: alias -> file path
	for _, node := range prog.Nodes {
		if imp, ok := node.(*ast.ImportStmt); ok {
			if imp.ModuleName != "" && imp.FilePath != "" {
				r.AliasToPath[imp.ModuleName] = imp.FilePath
				if r.Debug {
					fmt.Printf("[Resolver] Import alias: %s -> %s\n", imp.ModuleName, imp.FilePath)
				}
			}
		}
	}
	// Cache the current module's symbol table
	if prog != nil && prog.FilePath != "" {
		modName := prog.FilePath
		r.ModuleTables[modName] = r.Symbols
	}
	for _, node := range prog.Nodes {
		r.resolveNode(node)
	}
	if r.Debug {
		fmt.Printf("[Resolver] Finished semantic analysis for %s\n", r.File)
	}
}

func (r *Resolver) resolveNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.VarDeclStmt:
		r.resolveVarDecl(n)
	case *ast.AssignmentStmt:
		r.resolveAssignment(n)
	case *ast.ExpressionStmt:
		r.resolveExpressionStmt(n)
	case *ast.Block:
		if r.Debug {
			fmt.Println("[Resolver] Entering new scope")
		}
		outer := r.Symbols
		r.Symbols = semantic.NewSymbolTable(outer)
		for _, sub := range n.Nodes {
			r.resolveNode(sub)
		}
		r.Symbols = outer
		if r.Debug {
			fmt.Println("[Resolver] Exiting scope")
		}
	}
}

func (r *Resolver) resolveVarDecl(stmt *ast.VarDeclStmt) {
	for i, v := range stmt.Variables {
		name := v.Identifier.Name
		kind := semantic.SymbolVar
		if stmt.IsConst {
			kind = semantic.SymbolConst
		}
		// Type checking: ensure explicit type exists if provided
		if v.ExplicitType != nil {
			typeName := string(v.ExplicitType.Type())
			sym, found := r.Symbols.Lookup(typeName)
			if !found || sym.Kind != semantic.SymbolType {
				r.Reports.Add(r.File, v.Identifier.Loc(), "unknown type: "+typeName, report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
			}
		}
		sym := &semantic.Symbol{Name: name, Kind: kind, Type: v.ExplicitType}
		err := r.Symbols.Declare(name, sym)
		if err != nil {
			// Redeclaration error
			r.Reports.Add(r.File, v.Identifier.Loc(), err.Error(), report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
		// Check initializer expression if present
		if i < len(stmt.Initializers) && stmt.Initializers[i] != nil {
			r.resolveExpr(stmt.Initializers[i])
		}
	}
}

func (r *Resolver) resolveAssignment(stmt *ast.AssignmentStmt) { // Check that all left-hand side variables are declared
	for _, lhs := range stmt.Left {
		if id, ok := lhs.(*ast.IdentifierExpr); ok {
			varSym, found := r.Symbols.Lookup(id.Name)
			if !found {
				r.Reports.Add(r.File, id.Loc(), "assignment to undeclared variable: "+id.Name, report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
			} else if varSym.Type != nil {
				// Type checking: ensure type exists for variable
				typeName := string(varSym.Type.(ast.DataType).Type())
				typeSym, found := r.Symbols.Lookup(typeName)
				if !found || typeSym.Kind != semantic.SymbolType {
					r.Reports.Add(r.File, id.Loc(), "unknown type for variable: "+typeName, report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
				}
			}
		} else {
			r.resolveExpr(lhs)
		}
	}
	// Check right-hand side expressions
	for _, rhs := range stmt.Right {
		r.resolveExpr(rhs)
	}
}

func (r *Resolver) resolveExpressionStmt(stmt *ast.ExpressionStmt) {
	if stmt.Expressions != nil {
		for _, expr := range *stmt.Expressions {
			r.resolveExpr(expr)
		}
	}
}

func (r *Resolver) resolveExpr(expr ast.Expression) {
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		if _, found := r.Symbols.Lookup(e.Name); !found {
			r.Reports.Add(r.File, e.Loc(), "undeclared variable: "+e.Name, report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
		}
	case *ast.BinaryExpr:
		r.resolveExpr(e.Left)
		r.resolveExpr(e.Right)
	case *ast.UnaryExpr:
		r.resolveExpr(e.Operand)
	case *ast.PrefixExpr:
		r.resolveExpr(e.Operand)
	case *ast.PostfixExpr:
		r.resolveExpr(e.Operand)
	case *ast.FunctionCallExpr:
		r.resolveExpr(e.Caller)
		for _, arg := range e.Arguments {
			r.resolveExpr(arg)
		}
	case *ast.FieldAccessExpr:
		r.resolveExpr(e.Object)
	case *ast.ScopeResolutionExpr:
		alias := e.Module.Name
		if r.Debug {
			fmt.Printf("[Resolver] Resolving module alias: %s\n", alias)
		}
		filePath, ok := r.AliasToPath[alias]
		if !ok {
			r.Reports.Add(r.File, e.Module.Loc(), "unknown module: "+alias, report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if r.Debug {
				fmt.Printf("[Resolver] Alias '%s' not found in import map\n", alias)
			}
			return
		}
		modAST := r.ctx.GetModule(ctx.LocalModuleKey(filePath))
		if modAST == nil {
			modAST = r.ctx.GetModule(ctx.RemoteModuleKey(filePath))
		}
		if modAST == nil {
			r.Reports.Add(r.File, e.Module.Loc(), "unknown module: "+alias, report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if r.Debug {
				fmt.Printf("[Resolver] Module file '%s' not found for alias '%s'\n", filePath, alias)
			}
			return
		}
		// Get or build the module's symbol table
		modTable, found := r.ModuleTables[filePath]
		if !found {
			modTable = semantic.NewSymbolTable(nil)
			for _, node := range modAST.Nodes {
				if v, ok := node.(*ast.VarDeclStmt); ok {
					for _, varDecl := range v.Variables {
						modTable.Declare(varDecl.Identifier.Name, &semantic.Symbol{Name: varDecl.Identifier.Name, Kind: semantic.SymbolVar, Type: varDecl.ExplicitType})
					}
				}
			}
			r.ModuleTables[filePath] = modTable
			if r.Debug {
				fmt.Printf("[Resolver] Built symbol table for module: %s\n", filePath)
			}
		}
		if _, found := modTable.Lookup(e.Identifier.Name); !found {
			r.Reports.Add(r.File, e.Identifier.Loc(), "undeclared symbol in module '"+alias+"': "+e.Identifier.Name, report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if r.Debug {
				fmt.Printf("[Resolver] Symbol '%s' not found in module '%s' (file: %s)\n", e.Identifier.Name, alias, filePath)
			}
		} else if r.Debug {
			fmt.Printf("[Resolver] Resolved '%s::%s' (file: %s)\n", alias, e.Identifier.Name, filePath)
		}
	}
	// Add more cases as needed for literals, etc.
}
