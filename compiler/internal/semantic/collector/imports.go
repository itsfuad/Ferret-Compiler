package collector

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/semantic/analyzer"
	"compiler/report"

	"fmt"
)

func collectSymbolsFromImport(collector *analyzer.AnalyzerNode, imp *ast.ImportStmt) {
	defer func() {
		if r := recover(); r != nil {
			collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), fmt.Sprintf("Panic while collecting symbols from import: %v", r), report.COLLECTOR_PHASE)
		}
	}()

	moduleKey := imp.ImportPath.Value

	if moduleKey == "" {
		return
	}

	// Get the current module
	currentModule, err := collector.Ctx.GetModule(collector.Program.ImportPath)
	if err != nil {
		collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), "Failed to get current module for import validation", report.COLLECTOR_PHASE)
		return
	}

	// Resolve the import path based on context
	// For local imports within remote modules, convert to full GitHub path

	//module must be parses and stored already
	module, err := collector.Ctx.GetModule(moduleKey)
	if err != nil {
		collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), fmt.Sprintf("Failed to get imported module: %s", err.Error()), report.COLLECTOR_PHASE)
		return
	}

	// Add import to current module's symbol table with validation
	alias := imp.Alias
	if err := currentModule.SymbolTable.AddImport(alias, moduleKey, module.SymbolTable); err != nil {
		collector.Ctx.Reports.AddSemanticError(
			collector.Program.FullPath,
			imp.Loc(),
			err.Error(),
			report.COLLECTOR_PHASE,
		)
		return
	}

	//if already analyzed don't analyze again
	if module.Phase >= modules.PHASE_COLLECTED {
		return
	}

	// Collect symbols from the imported module recursively
	CollectSymbols(&analyzer.AnalyzerNode{
		Ctx:     collector.Ctx,
		Program: module.AST,
	})
}
