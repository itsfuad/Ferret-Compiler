package typecheck

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"fmt"
)

func CheckProgram(r *analyzer.AnalyzerNode) {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, nil, "Failed to get current module: "+err.Error(), report.RESOLVER_PHASE)
		return
	}
	for _, node := range r.Program.Nodes {
		checkNode(r, node, currentModule)
	}
	if r.Debug {
		colors.GREEN.Printf("Type checked '%s'\n", r.Program.FullPath)
	}
}

func checkNode(r *analyzer.AnalyzerNode, node ast.Node, cm *ctx.Module) {
	switch n := node.(type) {
	case *ast.ImportStmt:
		checkImportStmt(r, n, cm)
	case *ast.FunctionDecl:
		//checkFunctionDecl(r, n, cm)
	case *ast.VarDeclStmt:
		checkVariableDeclaration(r, n, cm)
	case *ast.TypeDeclStmt:
		// No type checking for type declarations, they are resolved in the resolver phase
	default:
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, node.Loc(), fmt.Sprintf("Unsupported node type <%T> for type checking", n), report.TYPECHECK_PHASE)
	}
}
