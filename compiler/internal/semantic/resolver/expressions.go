package resolver

import (
	"fmt"

	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)

// resolveExpr handles expression resolution
func resolveExpr(r *analyzer.AnalyzerNode, expr ast.Expression) {
	if expr == nil {
		panic("resolveExpr called with nil expression")
	}
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		resolveIdentifierExpr(r, e)
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
		resolveFunctionCallExpr(r, e)
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
		resolveArrayLiterals(r, e)
	case *ast.StructLiteralExpr:
		resolveStructLiteralExpr(r, e)
	case *ast.IndexableExpr:
		resolveExpr(r, *e.Indexable)
		resolveExpr(r, *e.Index)
	case *ast.FunctionLiteral:
		resolveFunctionLiteral(r, e)
	default:
		fmt.Printf("[Resolver] Expression <%T> is not implemented yet\n", e)
	}
}

// resolveExpressionStmt handles expression statement resolution
func resolveExpressionStmt(r *analyzer.AnalyzerNode, stmt *ast.ExpressionStmt) {
	if stmt.Expressions != nil {
		for _, expr := range *stmt.Expressions {
			resolveExpr(r, expr)
		}
	}
}

// resolveIdentifierExpr handles identifier expression resolution
func resolveIdentifierExpr(r *analyzer.AnalyzerNode, iden *ast.IdentifierExpr) {
	module, moduleExists := r.Ctx.Modules[r.Program.ImportPath]
	if !moduleExists {
		fmt.Printf("[Resolver] Module not found for path: %s\n", r.Program.FullPath)
		return
	}

	if module.SymbolTable == nil {
		fmt.Printf("[Resolver] SymbolTable is nil for module: %s\n", r.Program.FullPath)
		return
	}

	if _, found := module.SymbolTable.Lookup(iden.Name); !found {
		r.Ctx.Reports.Add(r.Program.FullPath, iden.Loc(), "undeclared variable: "+iden.Name, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
	}
}

// resolveFunctionCallExpr handles function call expression resolution
func resolveFunctionCallExpr(r *analyzer.AnalyzerNode, expr *ast.FunctionCallExpr) {
	resolveExpr(r, *expr.Caller)
	for _, arg := range expr.Arguments {
		resolveExpr(r, arg)
	}
}

// resolveArrayLiterals handles array literal expression resolution
func resolveArrayLiterals(r *analyzer.AnalyzerNode, expr *ast.ArrayLiteralExpr) {
	// Resolve array elements
	for _, element := range expr.Elements {
		resolveExpr(r, element)
	}
}

// resolveStructLiteralExpr handles struct literal expression resolution
func resolveStructLiteralExpr(r *analyzer.AnalyzerNode, expr *ast.StructLiteralExpr) {
	// Resolve struct field values
	for _, field := range expr.Fields {
		if field.FieldValue != nil {
			resolveExpr(r, *field.FieldValue)
		}
	}
}

// resolveFunctionLiteral handles function literal expression resolution
func resolveFunctionLiteral(r *analyzer.AnalyzerNode, fn *ast.FunctionLiteral) {
	// Resolve function body
	if fn.Body != nil {
		for _, node := range fn.Body.Nodes {
			resolveASTNode(r, node)
		}
	}
}
