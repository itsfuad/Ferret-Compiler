package resolver

import (
	"ferret/compiler/internal/ctx"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic/analyzer"
	"fmt"
)

func resolveExpr(r *analyzer.AnalyzerNode, expr ast.Expression, cm *ctx.Module) {
	if expr == nil {
		panic("resolveExpr called with nil expression")
	}
	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		resolveIdentifier(r, e, cm)
	case *ast.BinaryExpr:
		resolveExpr(r, *e.Left, cm)
		resolveExpr(r, *e.Right, cm)
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
