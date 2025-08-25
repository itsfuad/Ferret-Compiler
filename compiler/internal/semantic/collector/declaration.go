package collector

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/symbol"
	"compiler/report"
)

func collectVariableSymbols(c *analyzer.AnalyzerNode, decl *ast.VarDeclStmt, cm *modules.Module) {
	for _, variable := range decl.Variables {
		if variable.Identifier.Name == "" {
			c.Ctx.Reports.AddSemanticError(c.Program.FullPath, variable.Identifier.Loc(), "Variable identifier cannot be empty", report.COLLECTOR_PHASE)
			continue
		}

		// Declare the variable symbol with placeholder type
		variableSymbol := symbol.NewSymbolWithLocation(variable.Identifier.Name, symbol.SymbolVar, nil, variable.Identifier.Loc())
		err := cm.SymbolTable.Declare(variable.Identifier.Name, variableSymbol)
		if err != nil {
			c.Ctx.Reports.AddCriticalError(c.Program.FullPath, variable.Identifier.Loc(), "Failed to declare variable symbol: "+err.Error(), report.COLLECTOR_PHASE)
			continue
		}
		if c.Debug {
			colors.GREEN.Printf("Declared variable symbol %q (incomplete) at %s\n", variable.Identifier.Name, variable.Identifier.Loc())
		}
	}
	// Collect initializers if any
	for _, initializer := range decl.Initializers {
		if initializer == nil {
			continue
		}
		collectSymbols(c, initializer, cm) // Collect symbols from the initializer expression
		if c.Debug {
			colors.GREEN.Printf("Collected symbols from initializer at %s\n", initializer.Loc())
		}
	}
}

func collectTypeSymbol(c *analyzer.AnalyzerNode, decl *ast.TypeDeclStmt, cm *modules.Module) {
	aliasName := decl.Alias.Name
	if aliasName == "" {
		c.Ctx.Reports.AddSemanticError(c.Program.FullPath, decl.Alias.Loc(), "Type alias name cannot be empty", report.COLLECTOR_PHASE)
		return
	}

	// Declare the type symbol with placeholder type
	typeSymbol := symbol.NewSymbolWithLocation(aliasName, symbol.SymbolType, nil, decl.Alias.Loc())

	err := cm.SymbolTable.Declare(aliasName, typeSymbol)
	if err != nil {
		c.Ctx.Reports.AddCriticalError(c.Program.FullPath, decl.Alias.Loc(), "Failed to declare type symbol: "+err.Error(), report.COLLECTOR_PHASE)
		return
	}
	if c.Debug {
		colors.GREEN.Printf("Declared type symbol %q (incomplete) at %s\n", aliasName, decl.Alias.Loc())
	}
}
