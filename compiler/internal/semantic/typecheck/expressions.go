package typecheck

import (
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/types"
	"fmt"
)

// evaluateExpressionType infers the semantic type from an AST expression
func evaluateExpressionType(r *analyzer.AnalyzerNode, expr ast.Expression, cm *ctx.Module) ctx.Type {
	if expr == nil {
		return nil
	}

	var resultType ctx.Type

	switch e := expr.(type) {
	// Literals
	case *ast.StringLiteral:
		resultType = &ctx.PrimitiveType{Name: types.STRING}
	case *ast.IntLiteral:
		resultType = &ctx.PrimitiveType{Name: types.INT32}
	case *ast.FloatLiteral:
		resultType = &ctx.PrimitiveType{Name: types.FLOAT64}
	case *ast.BoolLiteral:
		resultType = &ctx.PrimitiveType{Name: types.BOOL}
	case *ast.ByteLiteral:
		resultType = &ctx.PrimitiveType{Name: types.BYTE}

	// Complex expressions
	case *ast.IdentifierExpr:
		resultType = checkIdentifierType(e, cm)
	case *ast.BinaryExpr:
		resultType = checkBinaryExprType(r, e, cm)
	case *ast.ArrayLiteralExpr:
		resultType = checkArrayLiteralType(r, e, cm)
	case *ast.IndexableExpr:
		resultType = checkIndexableType(r, e, cm)
	case *ast.VarScopeResolution:
		resultType = checkImportedSymbolType(r, e, cm)

	default:
		// Unknown expression type
		resultType = nil
		r.Ctx.Reports.AddCriticalError(
			r.Program.FullPath,
			e.Loc(),
			fmt.Sprintf("Unsupported expression type <%T> for type inference", e),
			report.TYPECHECK_PHASE,
		)
	}

	return resultType
}
