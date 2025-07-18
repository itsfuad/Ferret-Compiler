package typecheck

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)

func CheckProgram(r *analyzer.AnalyzerNode) {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, nil, "Failed to get current module: "+err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
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
	case *ast.FunctionDecl:
		//checkFunctionDecl(r, n, cm)
	case *ast.VarDeclStmt:
		checkVariableDeclaration(r, n, cm)
	default:
		r.Ctx.Reports.Add(r.Program.FullPath, node.Loc(), "Unsupported node type for type checking", report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
	}
}