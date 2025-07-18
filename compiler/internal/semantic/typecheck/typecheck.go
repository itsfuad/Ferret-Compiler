package typecheck

import (
	"compiler/colors"
	"compiler/ctx"
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
	default:
		r.Ctx.Reports.AddSemanticError(r.Program.FullPath, node.Loc(), fmt.Sprintf("Unsupported node type <%T> for type checking", n), report.TYPECHECK_PHASE)
	}
}

func checkImportStmt(c *analyzer.AnalyzerNode, imp *ast.ImportStmt, cm *ctx.Module) {
	if imp.ImportPath.Value == "" {
		c.Ctx.Reports.AddSyntaxError(c.Program.FullPath, imp.Loc(), "Import module name cannot be empty", report.COLLECTOR_PHASE)
		return
	}

	//module must be parses and stored already
	module, err := c.Ctx.GetModule(imp.ImportPath.Value)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, imp.Loc(), "Failed to get imported module: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}

	// collect functions from the imported module
	anz := analyzer.NewAnalyzerNode(module.AST, c.Ctx, c.Debug)
	CheckProgram(anz)
	//cm.SymbolTable.Imports[imp.ModuleName] = module.SymbolTable
}
