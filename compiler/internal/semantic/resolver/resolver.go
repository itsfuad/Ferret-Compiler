package resolver

import (
	"fmt"
	"os"

	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)

func ResolveProgram(r *analyzer.AnalyzerNode) {
	for _, node := range r.Program.Nodes {
		resolveNode(r, node)
	}
	if r.Debug {
		colors.GREEN.Printf("Resolved '%s'\n", r.Program.FullPath)
	}
}

func resolveNode(r *analyzer.AnalyzerNode, node ast.Node) {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, r.Program.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
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
