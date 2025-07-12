package resolver

import (
	"fmt"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
)


func resolveTypeScopeResolution(r *analyzer.AnalyzerNode, expr *ast.TypeScopeResolution) {

	modulename := expr.Module.Name

	importModuleName, ok := r.Program.ModulenameToImportpath[modulename]
	if !ok {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.Module.Loc(), fmt.Sprintf("module '%s' not found", modulename), report.RESOLVER_PHASE).AddHint("Check if the module is imported correctly").SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Get the imported module's symbol table
	importModule, err := r.Ctx.GetModule(importModuleName)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.Module.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	// Extract the type name from the type node
	var typeName string
	if userType, ok := expr.TypeNode.(*ast.UserDefinedType); ok {
		typeName = string(userType.TypeName)
	} else {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.TypeNode.Loc(), "invalid type in scope resolution", report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Look up the type symbol in the imported module's symbol table
	symbol, found := importModule.SymbolTable.Lookup(typeName)
	if !found {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.TypeNode.Loc(), fmt.Sprintf("type '%s' not found in module '%s'", typeName, modulename), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Verify it's actually a type
	if symbol.Kind != semantic.SymbolType {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.TypeNode.Loc(), fmt.Sprintf("expected type but found variable '%s' in module '%s'", typeName, modulename), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}
}

func resolveVarScopeResolution(r *analyzer.AnalyzerNode, expr ast.VarScopeResolution) {
	modulename := expr.Module.Name

	importModuleName, ok := r.Program.ModulenameToImportpath[modulename]
	if !ok {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.Module.Loc(), fmt.Sprintf("module '%s' not found", modulename), report.RESOLVER_PHASE).AddHint("Check if the module is imported correctly").SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Get the imported module's symbol table
	importModule, err := r.Ctx.GetModule(importModuleName)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.Module.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	// Look up the variable symbol in the imported module's symbol table
	symbol, found := importModule.SymbolTable.Lookup(expr.Var.Name)
	if !found {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.Var.Loc(), fmt.Sprintf("variable '%s' not found in module '%s'", expr.Var.Name, modulename), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Verify it's actually a variable (not a type)
	if symbol.Kind == semantic.SymbolType {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.Var.Loc(), fmt.Sprintf("expected variable but found type '%s' in module '%s'", expr.Var.Name, modulename), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}
}
