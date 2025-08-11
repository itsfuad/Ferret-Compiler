package collector

import (
	"ferret/colors"
	"ferret/internal/frontend/ast"
	"ferret/internal/modules"
	"ferret/internal/semantic/analyzer"
	"ferret/report"

	"fmt"
)

func collectSymbolsFromImport(collector *analyzer.AnalyzerNode, imp *ast.ImportStmt) {
	defer func() {
		if r := recover(); r != nil {
			collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), fmt.Sprintf("Panic while collecting symbols from import: %v", r), report.COLLECTOR_PHASE)
		}
	}()

	// Get the current module
	currentModule, err := collector.Ctx.GetModule(collector.Ctx.FullPathToImportPath(collector.Program.FullPath))
	if err != nil {
		collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), "Failed to get current module for import validation", report.COLLECTOR_PHASE)
		return
	}

	// Resolve the import path based on context
	// For local imports within remote modules, convert to full GitHub path
	moduleKey := modules.ResolveImportPath(imp.ImportPath.Value, collector.Program.FullPath, collector.Ctx.RemoteCachePath)
	colors.BLUE.Sprintf("moduleKey: %s", moduleKey)

	// ✅ SECURITY CHECK: Validate remote import permissions
	if err := modules.CheckCanImportRemoteModules(collector.Ctx.ProjectRootFullPath, moduleKey); err != nil {
		collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), err.Error(), report.COLLECTOR_PHASE)
		return
	}

	//module must be parses and stored already
	module, err := collector.Ctx.GetModule(moduleKey)
	if err != nil {
		collector.Ctx.Reports.AddCriticalError(collector.Program.FullPath, imp.Loc(), fmt.Sprintf("Failed to get imported module: %s", err.Error()), report.COLLECTOR_PHASE)
		return
	}

	// Add import to current module's symbol table with validation
	alias := imp.ModuleName
	if err := currentModule.SymbolTable.AddImport(alias, moduleKey, module.SymbolTable); err != nil {
		collector.Ctx.Reports.AddSemanticError(
			collector.Program.FullPath,
			imp.Loc(),
			err.Error(),
			report.COLLECTOR_PHASE,
		)
		return
	}

	if collector.Debug {
		colors.GREEN.Printf("Added import '%s' with alias '%s' to module '%s'\n", moduleKey, alias, collector.Ctx.FullPathToImportPath(collector.Program.FullPath))
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
