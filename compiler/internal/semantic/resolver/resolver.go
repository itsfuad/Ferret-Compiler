package resolver

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic/analyzer"
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
				colors.TEAL.Printf("Skipping resolution for '%s' (already in phase: %s)\n", r.Program.FullPath, currentPhase)
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

	// Track used imports for this specific file/module (reset for each file)
	r.UsedImports = make(map[string]bool)

	for _, node := range r.Program.Nodes {
		resolveNode(r, node, currentModule)
	}

	// Check for unused imports and report warnings
	checkUnusedImports(r, currentModule)

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
		colors.PINK.Printf("Resolving function declaration '%s' at %s\n", n.Identifier.Name, n.Loc())
		resolveFunctionDecl(r, n, cm)
	case *ast.MethodDecl:
		colors.PINK.Printf("Resolving method declaration '%s' at %s\n", n.Method.Name, n.Loc())
		resolveMethodDecl(r, n, cm)
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

// checkUnusedImports compares imported modules vs used modules and reports warnings
func checkUnusedImports(r *analyzer.AnalyzerNode, currentModule *modules.Module) {
	if r.Debug {
		colors.YELLOW.Printf("Checking unused imports. Used imports: %v\n", r.UsedImports)
	}

	// Collect all imports from the AST
	for _, node := range r.Program.Nodes {
		if importStmt, ok := node.(*ast.ImportStmt); ok {
			alias := importStmt.ModuleName
			if r.Debug {
				colors.YELLOW.Printf("Found import '%s' (alias: %s), used: %t\n", importStmt.ImportPath.Value, alias, r.UsedImports[alias])
			}
			if !r.UsedImports[alias] {
				r.Ctx.Reports.AddWarning(
					r.Program.FullPath,
					importStmt.Loc(),
					fmt.Sprintf("Unused import: '%s'", importStmt.ImportPath.Value),
					report.RESOLVER_PHASE,
				).AddHint("Remove the import or use symbols from this module")
			}
		}
	}
}

func resolveExpressionStmt(r *analyzer.AnalyzerNode, n *ast.ExpressionStmt, cm *modules.Module) {
	for _, expr := range *n.Expressions {
		resolveExpr(r, expr, cm)
	}
}
