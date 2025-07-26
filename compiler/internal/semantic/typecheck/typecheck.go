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
	importPath := r.Program.ImportPath

	// Check if this module can be processed for type checking phase
	if !r.Ctx.CanProcessPhase(importPath, ctx.PhaseTypeChecked) {
		currentPhase := r.Ctx.GetModulePhase(importPath)
		if currentPhase >= ctx.PhaseTypeChecked {
			// Already processed, skip
			if r.Debug {
				colors.GREEN.Printf("Skipping type checking for '%s' (already in phase: %s)\n", r.Program.FullPath, currentPhase.String())
			}
			return
		}
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, nil, "Module not ready for type checking phase", report.TYPECHECK_PHASE)
		return
	}

	currentModule, err := r.Ctx.GetModule(importPath)
	if err != nil {
		r.Ctx.Reports.AddCriticalError(r.Program.FullPath, nil, "Failed to get current module: "+err.Error(), report.TYPECHECK_PHASE)
		return
	}

	for _, node := range r.Program.Nodes {
		checkNode(r, node, currentModule)
	}

	// Mark module as type checked
	r.Ctx.SetModulePhase(importPath, ctx.PhaseTypeChecked)

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
	case *ast.AssignmentStmt:
		checkAssignmentStmt(r, n, cm)
	case *ast.ExpressionStmt:
		checkExprListTypeWithContext(r, n.Expressions, cm, true) // Allow void in expression statements
	default:
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, node.Loc(), fmt.Sprintf("Unsupported node type <%T> for type checking", n), report.TYPECHECK_PHASE)
	}
}
