package typecheck

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/semantic/analyzer"
)

// CheckProgram performs type checking on the entire program
func CheckProgram(r *analyzer.AnalyzerNode) {
	for _, node := range r.Program.Nodes {
		checkNode(r, node)
	}
	if r.Debug {
		colors.GREEN.Printf("Type checked '%s'\n", r.Program.FullPath)
	}
}

// checkNode performs type checking on a single AST node
func checkNode(r *analyzer.AnalyzerNode, node ast.Node) {
	switch n := node.(type) {
	case *ast.ImportStmt:
		checkImportStmt(r, n)
	case *ast.VarDeclStmt:
		checkVarDecl(r, n)
	case *ast.AssignmentStmt:
		checkAssignment(r, n)
	case *ast.ExpressionStmt:
		checkExpressionStmt(r, n)
	case *ast.TypeDeclStmt:
		checkTypeDecl(r, n)
	// Add more cases as needed
	default:
		// Skip nodes that don't need type checking
	}
}
