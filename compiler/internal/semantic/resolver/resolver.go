package resolver

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
)

// ResolveProgram is the main entry point for the resolver phase
func ResolveProgram(r *analyzer.AnalyzerNode) {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, nil, "Failed to get current module: "+err.Error(), report.RESOLVER_PHASE)
		return
	}
	for _, node := range r.Program.Nodes {
		resolveNode(r, node, currentModule)
	}
	if r.Debug {
		colors.GREEN.Printf("Resolved '%s'\n", r.Program.FullPath)
	}
}

// resolveNode dispatches resolution to the appropriate handler based on node type
func resolveNode(r *analyzer.AnalyzerNode, node ast.Node, cm *ctx.Module) {
	switch n := node.(type) {
	case *ast.FunctionDecl:
		resolveFunctionDecl(r, n, cm)
	case *ast.VarDeclStmt:
		resolveVariableDeclaration(r, n, cm)
	}
}
