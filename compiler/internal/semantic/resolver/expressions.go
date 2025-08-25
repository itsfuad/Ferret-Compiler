package resolver

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/symbol"
	"compiler/report"
	"fmt"
)

func resolveExpr(r *analyzer.AnalyzerNode, expr ast.Expression, cm *modules.Module) {
	fmt.Printf("Resolving expression of type: %T\n", expr)
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
		// Resolve the caller expression
		resolveExpr(r, *e.Caller, cm)
		// Resolve arguments
		for _, arg := range e.Arguments {
			resolveExpr(r, arg, cm)
		}
	case *ast.FieldAccessExpr:
		resolveExpr(r, *e.Object, cm)
	case *ast.VarScopeResolution:
		resolveImportedSymbol(r, e, cm)
	case *ast.SpreadExpr:
		resolveExpr(r, *e.Expression, cm)
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
		resolveArrayLiteral(r, e, cm)
	case *ast.StructLiteralExpr:
		resolveStructLiteral(r, e, cm)
	case *ast.FunctionLiteral:
		resolveFunctionLiteral(r, e, cm)
	case *ast.IndexableExpr:
		resolveExpr(r, *e.Indexable, cm)
		resolveExpr(r, *e.Index, cm)
	case *ast.CastExpr:
		resolveExpr(r, *e.Value, cm)
	default:
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, expr.Loc(), fmt.Sprintf("Expression <%T> is not implemented yet", e), report.RESOLVER_PHASE)
	}
}

func resolveStructLiteral(r *analyzer.AnalyzerNode, structLit *ast.StructLiteralExpr, cm *modules.Module) {
	for _, field := range structLit.Fields {
		resolveExpr(r, *field.FieldValue, cm)
	}
}

func resolveArrayLiteral(r *analyzer.AnalyzerNode, arrLit *ast.ArrayLiteralExpr, cm *modules.Module) {
	for _, elem := range arrLit.Elements {
		resolveExpr(r, elem, cm)
	}
}

func resolveIdentifier(r *analyzer.AnalyzerNode, id *ast.IdentifierExpr, cm *modules.Module) {
	sm, found := cm.SymbolTable.Lookup(id.Name)
	if !found {
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, id.Loc(), "undefined symbol: "+id.Name, report.RESOLVER_PHASE).AddHint("Did you forget to declare it or import the module where it's declared?")
		return
	}

	// For variables: check if they're used before declaration (forward reference)
	// For functions: allow forward references
	if sm.Kind == symbol.SymbolVar && sm.Location != nil {
		usagePos := id.Loc().Start
		declarationPos := sm.Location.Start

		// If variable is used before it's declared, that's an error
		if usagePos.Line < declarationPos.Line ||
			(usagePos.Line == declarationPos.Line && usagePos.Column < declarationPos.Column) {
			r.Ctx.Reports.AddSemanticError(
				r.Program.FullPath,
				id.Loc(),
				fmt.Sprintf("Cannot use variable %q before it is declared",
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
