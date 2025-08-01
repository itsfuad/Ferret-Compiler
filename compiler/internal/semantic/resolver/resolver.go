package resolver

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/semantic/analyzer"
	"fmt"
)

// ResolveProgram is the main entry point for the resolver phase
func ResolveProgram(r *analyzer.AnalyzerNode) {
	importPath := r.Program.ImportPath

	// Check if this module can be processed for resolution phase
	if !r.Ctx.CanProcessPhase(importPath, modules.PHASE_RESOLVED) {
		currentPhase := r.Ctx.GetModulePhase(importPath)
		if currentPhase >= modules.PHASE_RESOLVED {
			// Already processed or in a later phase, skip
			if r.Debug {
				colors.TEAL.Printf("Skipping resolution for '%s' (already in phase: %s)\n", r.Program.FullPath, currentPhase.String())
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
	r.Ctx.SetModulePhase(importPath, modules.PHASE_RESOLVED)

	if r.Debug {
		colors.GREEN.Printf("Resolved '%s'\n", r.Program.FullPath)
	}
}

// resolveNode dispatches resolution to the appropriate handler based on node type
func resolveNode(r *analyzer.AnalyzerNode, node ast.Node, cm *modules.Module) {
	switch n := node.(type) {
	case *ast.ImportStmt:
		resolveImportStmt(r, n, cm)
	case *ast.FunctionDecl:
		colors.PINK.Printf("Resolving function declaration '%s' at %s\n", n.Identifier.Name, n.Loc().String())
		resolveFunctionDecl(r, n, cm)
	case *ast.VarDeclStmt:
		resolveVariableDeclaration(r, n, cm)
	case *ast.TypeDeclStmt:
		resolveTypeDeclaration(r, n, cm)
	case *ast.AssignmentStmt:
		resolveAssignmentStmt(r, n, cm)
	case *ast.IfStmt:
		resolveIfStmt(r, n, cm)
	case *ast.Block:
		resolveBlock(r, n, cm)
	case *ast.ReturnStmt:
		resolveReturnStmt(r, n, cm)
	case *ast.ExpressionList:
		resolveExpressionList(r, n, cm)
	case *ast.ExpressionStmt:
		resolveExpressionStmt(r, n, cm)
	case *ast.FunctionLiteral:
		resolveFunctionLiteral(r, n, cm)
	default:
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, node.Loc(), fmt.Sprintf("Unsupported node type <%T> for resolution", n), report.RESOLVER_PHASE)
	}
}

func resolveExpressionStmt(r *analyzer.AnalyzerNode, n *ast.ExpressionStmt, cm *modules.Module) {
	for _, expr := range *n.Expressions {
		resolveExpr(r, expr, cm)
	}
}
