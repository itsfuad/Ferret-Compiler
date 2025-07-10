package parser

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"compiler/internal/utils/fs"
	"fmt"
	"path/filepath"
	"strings"
)

// parseImport parses an import statement
func parseImport(p *Parser) ast.Node {
	start := p.consume(lexer.IMPORT_TOKEN, report.EXPECTED_IMPORT_KEYWORD)
	importPath := p.consume(lexer.STRING_TOKEN, report.EXPECTED_IMPORT_PATH)

	importpath := importPath.Value

	// Support: import "path" as Alias;
	var moduleName string
	if p.match(lexer.AS_TOKEN) {
		p.advance() // consume 'as'
		aliasToken := p.consume(lexer.IDENTIFIER_TOKEN, "Expected identifier after 'as' in import")
		moduleName = aliasToken.Value
	} else {
		// Default: use last part of path (without extension)
		parts := strings.Split(importpath, "/")
		if len(parts) == 0 {
			p.ctx.Reports.Add(p.fullPath, source.NewLocation(&start.Start, &importPath.End), report.INVALID_IMPORT_PATH, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		sufs := strings.Split(parts[len(parts)-1], ".")
		suf := "." + sufs[len(sufs)-1]
		moduleName = strings.TrimSuffix(parts[len(parts)-1], suf)
	}

	loc := *source.NewLocation(&start.Start, &importPath.End)

	// Use fs.ResolveModule to get the full path
	moduleFullPath, _, err := fs.ResolveModule(importpath, p.fullPath, p.ctx, false)
	if err != nil {
		p.ctx.Reports.Add(p.fullPath, &loc, err.Error(), report.PARSING_PHASE).SetLevel(report.CRITICAL_ERROR)
		return nil
	}

	isRemote := fs.IsRemote(importpath)

	stmt := &ast.ImportStmt{
		ImportPath: &ast.StringLiteral{
			Value:    importpath,
			Location: loc,
		},
		ModuleName: moduleName,
		FullPath:   moduleFullPath,
		IsRemote:   isRemote,
		Location:   loc,
	}

	// Add dependency edge and check for cycles,
	p.ctx.AddDepEdge(p.fullPath, moduleFullPath)

	// Always start cycle detection from the entrypoint
	entryRel := p.ctx.EntryPoint
	if p.ctx.RootDir != "" {
		entryRel = filepath.ToSlash(filepath.Join("", p.ctx.EntryPoint))
	}
	entryKey := entryRel
	if cycle, found := p.ctx.DetectCycle(entryKey); found {
		cycleStr := strings.Join(cycle, " -> ")
		msg := "Circular import detected: " + cycleStr
		p.ctx.Reports.Add(p.fullPath, &loc, msg, report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
		colors.RED.Println(msg)
		return stmt
	}

	// Check if the module is already cached
	if !p.ctx.HasModule(importpath) {

		module := NewParser(moduleFullPath, p.ctx, p.debug).Parse()

		if module == nil {
			p.ctx.Reports.Add(p.fullPath, &loc, "Failed to parse imported module", report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
			return &ast.ImportStmt{Location: loc}
		}

		p.ctx.AddModule(importpath, module)
		colors.GREEN.Printf("Cached <- Module '%s'\n", importpath)
	} else {
		colors.ORANGE.Printf("Skipping module '%s' : Already cached\n", importpath)
	}

	p.ctx.AliasToModuleName[moduleName] = importpath

	fmt.Printf("Parsing import: %s -> %s\n", p.ctx.FullPathToModuleName(p.fullPath), importpath)

	return stmt
}

func parseScopeResolution(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	// Handle scope resolution operator
	if module, ok := expr.(*ast.IdentifierExpr); ok {
		p.consume(lexer.SCOPE_TOKEN, report.EXPECTED_SCOPE_RESOLUTION_OPERATOR)
		if !p.match(lexer.IDENTIFIER_TOKEN) {
			token := p.peek()
			p.ctx.Reports.Add(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected identifier after '::'", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil, false
		}
		member := parseIdentifier(p)
		return &ast.VarScopeResolution{
			Module:   module,
			Var:      member,
			Location: *source.NewLocation(module.Loc().Start, member.Loc().End),
		}, true
	} else {
		token := p.peek()
		p.ctx.Reports.Add(p.fullPath, source.NewLocation(&token.Start, &token.End), "Left side of '::' must be an identifier", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}
}
