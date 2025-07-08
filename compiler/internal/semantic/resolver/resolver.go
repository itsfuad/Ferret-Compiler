package resolver

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"fmt"
	"os"
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

	// Build import alias map: alias -> file path
	for _, node := range r.program.Nodes {
		if imp, ok := node.(*ast.ImportStmt); ok {
			if imp.ModuleName != "" && imp.FilePath != "" {
				if r.Debug {
					fmt.Printf("[Resolver] Import alias: %s -> %s\n", imp.ModuleName, imp.FilePath)
				}
				modulePath := imp.FilePath
				colors.PURPLE.Printf("Root dir: %s\n", r.ctx.RootDir)
				colors.PURPLE.Printf("Got module key: %s for module: %s\n", modulePath, modulePath)
				modAST := r.ctx.GetModule(modulePath).AST
				colors.PURPLE.Printf("Imported module: %s\n", modAST.FilePath)
				if modAST == nil {
					modAST = r.ctx.GetModule(modulePath).AST
				}
				if modAST != nil {
					colors.PURPLE.Printf("Searching for module: %s\n", modulePath)
					module, found := r.ctx.Modules[modulePath]
					if !found {
						r.ctx.Reports.Add(r.program.FilePath, imp.Loc(), "module not found: "+modulePath, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
					}
					semantic.AddPreludeSymbols(module.SymbolTable)
					resolver := NewResolver(modAST, r.ctx, r.Debug)
					resolver.ResolveProgram()
				}
			}
		}
	}

	for _, node := range r.program.Nodes {
		fmt.Printf("[Resolver] Resolving node: %T\n", node)
		r.resolveNode(node)
	}
	if r.Debug {
		fmt.Printf("[Resolver] Finished semantic analysis for %s\n", r.program.FilePath)
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
		outer := r.ctx.Modules[r.program.FilePath].SymbolTable
		r.ctx.Modules[r.program.FilePath].SymbolTable = semantic.NewSymbolTable(outer)
		for _, sub := range n.Nodes {
			r.resolveNode(sub)
		}
		r.ctx.Modules[r.program.FilePath].SymbolTable = outer
		if r.Debug {
			fmt.Println("[Resolver] Exiting scope")
		}
	default:
		fmt.Printf("[Resolver] Node <%T> is not implemented yet\n", n)
		os.Exit(-1)
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
			sym, found := r.ctx.Modules[r.program.FilePath].SymbolTable.Lookup(typeName)
			if !found || sym.Kind != semantic.SymbolType {
				r.ctx.Reports.Add(r.program.FilePath, v.Identifier.Loc(), "unknown type: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			}
		}
		sym := &semantic.Symbol{Name: name, Kind: kind, Type: v.ExplicitType}
		//err := r.ctx.Modules[r.program.FilePath].SymbolTable.Declare(name, sym)
		mod := r.ctx.Modules[r.program.FilePath]
		if mod == nil {
			r.ctx.Reports.Add(r.program.FilePath, v.Identifier.Loc(), "module not found: "+r.program.FilePath, report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		}
		err := mod.SymbolTable.Declare(name, sym)
		if err != nil {
			// Redeclaration error
			r.ctx.Reports.Add(r.program.FilePath, v.Identifier.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
			varSym, found := r.ctx.Modules[r.program.FilePath].SymbolTable.Lookup(id.Name)
			if !found {
				r.ctx.Reports.Add(r.program.FilePath, id.Loc(), "assignment to undeclared variable: "+id.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			} else if varSym.Type != nil {
				// Type checking: ensure type exists for variable
				typeName := string(varSym.Type.(ast.DataType).Type())
				typeSym, found := r.ctx.Modules[r.program.FilePath].SymbolTable.Lookup(typeName)
				if !found || typeSym.Kind != semantic.SymbolType {
					r.ctx.Reports.Add(r.program.FilePath, id.Loc(), "unknown type for variable: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
		if _, found := r.ctx.Modules[r.program.FilePath].SymbolTable.Lookup(e.Name); !found {
			r.ctx.Reports.Add(r.program.FilePath, e.Loc(), "undeclared variable: "+e.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
		filePath, ok := r.ctx.AliasToPath[alias]
		if !ok {
			r.ctx.Reports.Add(r.program.FilePath, e.Module.Loc(), "unknown module: "+alias, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if r.Debug {
				fmt.Printf("[Resolver] Alias '%s' not found in import map\n", alias)
			}
			return
		}
		modTable, found := r.ctx.Modules[filePath]
		if !found {
			panic(fmt.Sprintf("Module table for %s not found during scope resolution", filePath))
		}
		// Link the imported module's symbol table in the current module's Imports map (idempotent)
		r.ctx.Modules[r.program.FilePath].SymbolTable.Imports[alias] = modTable.SymbolTable
		if _, found := modTable.SymbolTable.Lookup(e.Identifier.Name); !found {
			r.ctx.Reports.Add(r.program.FilePath, e.Identifier.Loc(), "undeclared symbol in module '"+alias+"': "+e.Identifier.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if r.Debug {
				fmt.Printf("[Resolver] Symbol '%s' not found in module '%s' (file: %s)\n", e.Identifier.Name, alias, filePath)
			}
		} else if r.Debug {
			fmt.Printf("[Resolver] Resolved '%s::%s' (file: %s)\n", alias, e.Identifier.Name, filePath)
		}
	default:
		fmt.Printf("[Resolver] Expression <%T> is not implemented yet\n", e)
		os.Exit(-1)
	}
	// Add more cases as needed for literals, etc.
}
