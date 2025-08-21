package parser

import (
	"fmt"
	"strings"

	"compiler/config"
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

	skip := false

	stmt := &ast.ImportStmt{
		ImportPath: &ast.StringLiteral{
			Value:    importpath,
			Location: loc,
		},
		Alias:    alias,
		Location: loc,
	}

	config, moduleFullPath, modType, err := p.ctx.ResolveImportPath(importpath)
	if err != nil {
		p.ctx.Reports.AddError(p.fullPath, &loc, err.Error(), report.PARSING_PHASE)
		skip = true
	}

	if !skip {
		skip = shouldSkip(p, &loc, config, importpath, modType)
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

	stmt.LocationOnDisk = moduleFullPath

	return stmt
}

func shouldSkip(p *Parser, loc *source.Location, config *config.ProjectConfig, importPath string, modType modules.ModuleType) bool {
	if modType != modules.LOCAL && modType != modules.BUILTIN {
		con := p.ctx.ProjectStack.Peek()
		if modType == modules.NEIGHBOR && !con.External.AllowExternalImport {
			p.ctx.Reports.AddError(p.fullPath, loc, fmt.Sprintf("Cannot import neighbor module %q as your project disabled neighbor project access", importPath), report.PARSING_PHASE).AddHint("Enable allow-external-import=true in <project_root>/fer.ret")
			return true
		} else if modType == modules.REMOTE && !con.External.AllowRemoteImport {
			p.ctx.Reports.AddError(p.fullPath, loc, fmt.Sprintf("Cannot import remote module %q as your project disabled remote imports", importPath), report.PARSING_PHASE).AddHint("Enable allow-remote-import=true in <project_root>/fer.ret")
			return true
		}
	}
	if modType == modules.BUILTIN && !config.External.AllowSharing {
		p.ctx.Reports.AddError(p.fullPath, loc, fmt.Sprintf("Module %q is not enabled for sharing", importPath), report.PARSING_PHASE)
		return true
	}
	return false
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
