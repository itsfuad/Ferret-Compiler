package resolver

import (
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"fmt"
)

// ResolveProgram is the main entry point for the resolver phase
func ResolveProgram(r *analyzer.AnalyzerNode) {
	importPath := r.Program.ImportPath

	// Check if this module can be processed for resolution phase
	if !r.Ctx.CanProcessPhase(importPath, ctx.PhaseResolved) {
		currentPhase := r.Ctx.GetModulePhase(importPath)
		if currentPhase >= ctx.PhaseResolved {
			// Already processed or in a later phase, skip
			if r.Debug {
				colors.GREEN.Printf("Skipping resolution for '%s' (already in phase: %s)\n", r.Program.FullPath, currentPhase.String())
			}
			return
		}
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, nil, "Module not ready for resolution phase", report.RESOLVER_PHASE)
		return
	}

	currentModule, err := r.Ctx.GetModule(importPath)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, nil, "Failed to get current module: "+err.Error(), report.RESOLVER_PHASE)
		return
	}

	for _, node := range r.Program.Nodes {
		resolveNode(r, node, currentModule)
	}

	// Mark module as resolved
	r.Ctx.SetModulePhase(importPath, ctx.PhaseResolved)

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
	case *ast.TypeDeclStmt:
		resolveTypeDeclaration(r, n, cm)
	case *ast.AssignmentStmt:
		resolveAssignmentStmt(r, n, cm)
	case *ast.Block:
		//pass
	case *ast.ExpressionList:
		resolveExpressionList(r, n, cm)
	case *ast.ExpressionStmt:
		resolveExpressionStmt(r, n, cm)
	default:
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, node.Loc(), fmt.Sprintf("Unsupported node type <%T> for resolution", n), report.RESOLVER_PHASE)
	}
}

func resolveExpressionStmt(r *analyzer.AnalyzerNode, n *ast.ExpressionStmt, cm *ctx.Module) {
	colors.CYAN.Printf("Resolving expression statement: %v\n", n.Expressions)

	for _, expr := range *n.Expressions {
		resolveExpr(r, expr, cm)
	}
}
