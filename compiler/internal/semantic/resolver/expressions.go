package resolver

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	symbolpkg "compiler/internal/symbol"
	"fmt"
)

func resolveExpr(r *analyzer.AnalyzerNode, expr ast.Expression, cm *modules.Module) {
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
	case *ast.CastExpr:
		// Resolve the value being cast
		resolveExpr(r, *e.Value, cm)
		// Target type doesn't need resolution as it's a type declaration
	default:
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, expr.Loc(), fmt.Sprintf("Expression <%T> is not implemented yet", e), report.RESOLVER_PHASE)
	}
}

func resolveIdentifier(r *analyzer.AnalyzerNode, id *ast.IdentifierExpr, cm *modules.Module) {
	symbol, found := cm.SymbolTable.Lookup(id.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, id.Loc(), "undefined symbol: "+id.Name, report.RESOLVER_PHASE)
		return
	}

	// For variables: check if they're used before declaration (forward reference)
	// For functions: allow forward references
	if symbol.Kind == symbolpkg.SymbolVar && symbol.Location != nil {
		usagePos := id.Loc().Start
		declarationPos := symbol.Location.Start

		// If variable is used before it's declared, that's an error
		if usagePos.Line < declarationPos.Line ||
			(usagePos.Line == declarationPos.Line && usagePos.Column < declarationPos.Column) {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				id.Loc(),
				fmt.Sprintf("Cannot use variable '%s' before it is declared",
					id.Name),
				report.RESOLVER_PHASE,
			)
		}
	}
	// Functions can be called before declaration - no check needed
}

func resolveExpressionList(r *analyzer.AnalyzerNode, exprList *ast.ExpressionList, cm *modules.Module) {
	for _, expr := range *exprList {
		resolveExpr(r, expr, cm)
	}
}
