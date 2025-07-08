package parser

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"compiler/internal/utils/fs"
	"path/filepath"
	"strings"
)

// parseImport parses an import statement
func parseImport(p *Parser) ast.Node {
	start := p.consume(lexer.IMPORT_TOKEN, report.EXPECTED_IMPORT_KEYWORD)
	importPath := p.consume(lexer.STRING_TOKEN, report.EXPECTED_IMPORT_PATH)

	// Support: import "path" as Alias;
	var moduleName string
	if p.match(lexer.AS_TOKEN) {
		p.advance() // consume 'as'
		aliasToken := p.consume(lexer.IDENTIFIER_TOKEN, "Expected identifier after 'as' in import")
		moduleName = aliasToken.Value
	} else {
		// Default: use last part of path (without extension)
		parts := strings.Split(importPath.Value, "/")
		if len(parts) == 0 {
			p.ctx.Reports.Add(p.filePath, source.NewLocation(&start.Start, &importPath.End), report.INVALID_IMPORT_PATH, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		sufs := strings.Split(parts[len(parts)-1], ".")
		suf := "." + sufs[len(sufs)-1]
		moduleName = strings.TrimSuffix(parts[len(parts)-1], suf)
	}

	loc := *source.NewLocation(&start.Start, &importPath.End)

	// Determine logical import path of the importer
	var importerLogicalPath string
	if strings.HasPrefix(p.filePath, p.ctx.CachePath) {
		// This is a cached remote file, so get the remote import path from the cache path
		// Remove the cache prefix and convert to github.com/... form
		rel, _ := filepath.Rel(p.ctx.CachePath, p.filePath)
		importerLogicalPath = filepath.ToSlash(rel)
	} else {
		// Local file: use project-relative path
		rel, _ := filepath.Rel(p.ctx.RootDir, p.filePath)
		importerLogicalPath = filepath.ToSlash(rel)
	}

	// Use fs.ResolveModule to get the absolute path
	absPath, moduleKey, err := fs.ResolveModule(importPath.Value, p.filePath, importerLogicalPath, p.ctx, false)
	// Convert absPath to project-root relative, then normalize to slashes
	relPath, _ := filepath.Rel(p.ctx.RootDir, absPath)
	relPath = filepath.ToSlash(relPath)

	stmt := &ast.ImportStmt{
		ImportPath: &ast.StringLiteral{
			Value:    importPath.Value,
			Location: loc,
		},
		ModuleName: moduleName,
		FilePath:   relPath,
		Location:   loc,
	}

	if err != nil {
		p.ctx.Reports.Add(p.filePath, &loc, err.Error(), report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return stmt
	}

	// Add dependency edge and check for cycles
	importerKey := ctx.ModuleKey{IsRemote: strings.HasPrefix(importerLogicalPath, "github.com/"), Path: importerLogicalPath}.String()
	importedKey := moduleKey.String()
	p.ctx.AddDepEdge(importerKey, importedKey)

	// Always start cycle detection from the entrypoint
	entryRel := p.ctx.EntryPoint
	if p.ctx.RootDir != "" {
		entryRel = filepath.ToSlash(filepath.Join("", p.ctx.EntryPoint))
	}
	entryKey := ctx.ModuleKey{IsRemote: false, Path: entryRel}.String()
	if cycle, found := p.ctx.DetectCycle(entryKey); found {
		cycleStr := strings.Join(cycle, " -> ")
		msg := "Circular import detected: " + cycleStr
		p.ctx.Reports.Add(p.filePath, &loc, msg, report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
		colors.RED.Println(msg)
		return stmt
	}

	// Check if the module is already cached
	if !p.ctx.HasModule(moduleKey) {
		module := NewParser(absPath, p.ctx, p.debug).Parse()

		if module == nil {
			p.ctx.Reports.Add(p.filePath, &loc, "Failed to parse imported module", report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
			return &ast.ImportStmt{Location: loc}
		}

		p.ctx.AddModule(moduleKey, module)
		colors.GREEN.Printf("Cached <- Module '%s'\n", moduleName)
	} else {
		colors.ORANGE.Printf("Skipping module '%s' : Already cached\n", moduleName)
	}

	return stmt
}

func parseScopeResolution(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	// Handle scope resolution operator
	if module, ok := expr.(*ast.IdentifierExpr); ok {
		p.consume(lexer.SCOPE_TOKEN, report.EXPECTED_SCOPE_RESOLUTION_OPERATOR)
		if !p.match(lexer.IDENTIFIER_TOKEN) {
			token := p.peek()
			p.ctx.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Expected identifier after '::'", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil, false
		}
		member := parseIdentifier(p)
		return &ast.ScopeResolutionExpr{
			Module:     module,
			Identifier: member,
			Location:   *source.NewLocation(module.Loc().Start, member.Loc().End),
		}, true
	} else {
		token := p.peek()
		p.ctx.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Left side of '::' must be an identifier", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}
}
