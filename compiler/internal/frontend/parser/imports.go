package parser

import (
	"fmt"
	"strings"

	"compiler/colors"
	//"compiler/config"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/modules"
	"compiler/internal/source"
	"compiler/internal/utils/fs"
	"compiler/report"
)

// parseImport parses an import statement
func parseImport(p *Parser) ast.Node {

	start := p.consume(lexer.IMPORT_TOKEN, report.EXPECTED_IMPORT_KEYWORD)
	importToken := p.consume(lexer.STRING_TOKEN, report.EXPECTED_IMPORT_PATH)

	importpath := importToken.Value

	// Support: import "path" as Alias;
	var alias string
	if p.match(lexer.AS_TOKEN) {
		p.advance() // consume 'as'
		aliasToken := p.consume(lexer.IDENTIFIER_TOKEN, "Expected identifier after 'as' in import")
		alias = aliasToken.Value
	} else {
		alias = fs.LastPart(importpath)
	}

	loc := *source.NewLocation(&start.Start, &importToken.End)

	moduleFullPath, config, modType, err := p.ctx.ImportPathToFullPath(importpath)

	colors.BOLD_PURPLE.Printf("Import module path %q from import path %q\n", moduleFullPath, importpath)

	
	stmt := &ast.ImportStmt{
		ImportPath: &ast.StringLiteral{
			Value:    importpath,
			Location: loc,
		},
		Alias:          alias,
		LocationOnDisk: moduleFullPath,
		Location:       loc,
	}

	skip := false
	
	if (modType != modules.LOCAL && modType != modules.BUILTIN) && !p.ctx.PeekProjectStack().Remote.Enabled {
		skip = true
		p.ctx.Reports.AddError(p.fullPath, &loc, fmt.Sprintf("Cannot import external module %q as your project disabled external project access", moduleFullPath), report.PARSING_PHASE).AddHint("Enable external project access to true in <project_root>/fer.ret")
	} else if modType != modules.LOCAL && !config.Remote.Share {
		skip = true
		p.ctx.Reports.AddError(p.fullPath, &loc, fmt.Sprintf("Module %q is not enabled for sharing", moduleFullPath), report.PARSING_PHASE)
	}


	if err != nil {
		p.ctx.Reports.AddError(p.fullPath, &loc, err.Error(), report.PARSING_PHASE)
		skip = true
	}

	if skip {
		stmt.ImportPath.Value = ""
		return stmt
	}

	// Check for circular dependency before adding the import
	if cycle, found := p.ctx.DetectCycle(p.importPath, importpath); found {
		cycleStr := strings.Join(cycle, " → ")

		currentModule := p.importPath
		targetModule := importpath

		cycleMsg := fmt.Sprintf("Import cycle detected: %s\n - %q cannot import %q (already in dependency path)",
			cycleStr, currentModule, targetModule)
		p.ctx.Reports.AddCriticalError(p.fullPath, &loc, cycleMsg, report.PARSING_PHASE)
		return stmt
	}

	// Check if the module is already parsed
	if !p.ctx.IsModuleParsed(importpath) {
		module := NewParserWithImportPath(moduleFullPath, importpath, p.ctx, p.debug).Parse()
		if module == nil {
			p.ctx.Reports.AddSemanticError(p.fullPath, &loc, "Failed to parse imported module", report.PARSING_PHASE)
			return &ast.ImportStmt{Location: loc}
		}
	}

	return stmt
}

func parseScopeResolution(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	// Handle scope resolution operator
	if module, ok := expr.(*ast.IdentifierExpr); ok {
		p.consume(lexer.SCOPE_TOKEN, report.EXPECTED_SCOPE_RESOLUTION_OPERATOR)
		if !p.match(lexer.IDENTIFIER_TOKEN) {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected identifier after '::'", report.PARSING_PHASE)
			return nil, false
		}
		member := parseIdentifier(p)
		return &ast.VarScopeResolution{
			Module:     module,
			Identifier: member,
			Location:   *source.NewLocation(module.Loc().Start, member.Loc().End),
		}, true
	}
	token := p.peek()
	p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Left side of '::' must be an identifier", report.PARSING_PHASE)
	return nil, false
}
