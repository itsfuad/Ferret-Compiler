package resolver

import (
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/source"
	"fmt"
)

// getSymbolKindName returns a human-readable name for a symbol kind
func getSymbolKindName(kind semantic.SymbolKind) string {
	switch kind {
	case semantic.SymbolVar:
		return "variable"
	case semantic.SymbolConst:
		return "constant"
	case semantic.SymbolType:
		return "type"
	case semantic.SymbolFunc:
		return "function"
	case semantic.SymbolStruct:
		return "struct"
	case semantic.SymbolField:
		return "field"
	default:
		return "unknown"
	}
}

// getImportedModule is a helper function to get an imported module by name
func getImportedModule(r *analyzer.AnalyzerNode, moduleName string, errorLoc *source.Location) (*ctx.Module, bool) {
	importModuleName, ok := r.Program.ModulenameToImportpath[moduleName]
	if !ok {
		r.Ctx.Reports.Add(r.Program.FullPath, errorLoc, fmt.Sprintf("module '%s' not found", moduleName), report.RESOLVER_PHASE).AddHint("Check if the module is imported correctly").SetLevel(report.SEMANTIC_ERROR)
		return nil, false
	}

	importModule, err := r.Ctx.GetModule(importModuleName)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, errorLoc, err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return nil, false
	}

	return importModule, true
}

// extractTypeName extracts the type name from a type node, with proper error handling
func extractTypeName(r *analyzer.AnalyzerNode, typeNode ast.DataType) (string, bool) {
	if userType, ok := typeNode.(*ast.UserDefinedType); ok {
		return string(userType.TypeName), true
	}

	r.Ctx.Reports.Add(r.Program.FullPath, typeNode.Loc(), "invalid type in scope resolution", report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
	return "", false
}

// validateSymbolInModule validates that a symbol exists in a module and has the expected kind
func validateSymbolInModule(r *analyzer.AnalyzerNode, module *ctx.Module, symbolName string, expectedKind semantic.SymbolKind, errorLoc *source.Location, moduleName string) bool {
	symbol, found := module.SymbolTable.Lookup(symbolName)
	if !found {
		r.Ctx.Reports.Add(r.Program.FullPath, errorLoc, fmt.Sprintf("%s '%s' not found in module '%s'", getSymbolKindName(expectedKind), symbolName, moduleName), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return false
	}

	if symbol.Kind != expectedKind {
		r.Ctx.Reports.Add(r.Program.FullPath, errorLoc, fmt.Sprintf("expected %s but found %s '%s' in module '%s'", getSymbolKindName(expectedKind), getSymbolKindName(symbol.Kind), symbolName, moduleName), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return false
	}

	return true
}

func resolveTypeScopeResolution(r *analyzer.AnalyzerNode, expr *ast.TypeScopeResolution) {
	// Get the imported module
	importModule, ok := getImportedModule(r, expr.Module.Name, expr.Module.Loc())
	if !ok {
		return // Error already reported by helper
	}

	// Extract type name and validate - reuse logic from existing resolvers
	typeName, ok := extractTypeName(r, expr.TypeNode)
	if !ok {
		return // Error already reported
	}

	// Verify the type exists in the specific imported module
	validateSymbolInModule(r, importModule, typeName, semantic.SymbolType, expr.TypeNode.Loc(), expr.Module.Name)
}

func resolveVarScopeResolution(r *analyzer.AnalyzerNode, expr ast.VarScopeResolution) {
	// Get the imported module
	importModule, ok := getImportedModule(r, expr.Module.Name, expr.Module.Loc())
	if !ok {
		return // Error already reported by helper
	}

	// Validate the variable exists in the specific imported module
	// Note: We validate it's NOT a type (variables can be const or var)
	symbol, found := importModule.SymbolTable.Lookup(expr.Var.Name)
	if !found {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.Var.Loc(), fmt.Sprintf("variable '%s' not found in module '%s'", expr.Var.Name, expr.Module.Name), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Verify it's actually a variable (not a type)
	if symbol.Kind == semantic.SymbolType {
		r.Ctx.Reports.Add(r.Program.FullPath, expr.Var.Loc(), fmt.Sprintf("expected variable but found type '%s' in module '%s'", expr.Var.Name, expr.Module.Name), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}
}
