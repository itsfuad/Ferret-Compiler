package resolver

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"fmt"
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
	colors.BRIGHT_BROWN.Printf("Resolving node of type <%T>\n", node)
	switch n := node.(type) {
	case *ast.ImportStmt:
		resolveImportStmt(r, n, cm)
	case *ast.FunctionDecl:
		resolveFunctionDecl(r, n, cm)
	case *ast.VarDeclStmt:
		resolveVariableDeclaration(r, n, cm)
	case *ast.ExpressionStmt:
		colors.CYAN.Printf("Resolving expression statement: %v\n", n.Expressions)
	default:
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, node.Loc(), fmt.Sprintf("Unsupported node type <%T> for resolution", n), report.RESOLVER_PHASE)
	}
}

func resolveImportStmt(r *analyzer.AnalyzerNode, imp *ast.ImportStmt, cm *ctx.Module) {
	if imp.ImportPath.Value == "" {
		r.Ctx.Reports.AddSyntaxError(r.Program.FullPath, imp.Loc(), "Import module name cannot be empty", report.COLLECTOR_PHASE)
		return
	}

	//module must be parses and stored already
	module, err := r.Ctx.GetModule(imp.ImportPath.Value)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, imp.Loc(), "Failed to get imported module: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}

	// collect functions from the imported module
	anz := analyzer.NewAnalyzerNode(module.AST, r.Ctx, r.Debug)
	ResolveProgram(anz)
	//cm.SymbolTable.Imports[imp.ModuleName] = module.SymbolTable
}