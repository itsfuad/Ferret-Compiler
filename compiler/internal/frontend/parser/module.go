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

	canonicalName := importPath.Value

	// Support: import "path" as Alias;
	var moduleName string
	if p.match(lexer.AS_TOKEN) {
		p.advance() // consume 'as'
		aliasToken := p.consume(lexer.IDENTIFIER_TOKEN, "Expected identifier after 'as' in import")
		moduleName = aliasToken.Value
	} else {
		// Default: use last part of path (without extension)
		parts := strings.Split(canonicalName, "/")
		if len(parts) == 0 {
			p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&start.Start, &importPath.End), report.INVALID_IMPORT_PATH, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		sufs := strings.Split(parts[len(parts)-1], ".")
		suf := "." + sufs[len(sufs)-1]
		moduleName = strings.TrimSuffix(parts[len(parts)-1], suf)
	}

	loc := *source.NewLocation(&start.Start, &importPath.End)

	// Use fs.ResolveModule to get the absolute path
	moduleAbsPath, _, err := fs.ResolveModule(canonicalName, p.filePathAbs, p.ctx, false)
	if err != nil {
		p.ctx.Reports.Add(p.filePathAbs, &loc, err.Error(), report.PARSING_PHASE).SetLevel(report.CRITICAL_ERROR)
		return nil
	}

	stmt := &ast.ImportStmt{
		ImportPath: &ast.StringLiteral{
			Value:    canonicalName,
			Location: loc,
		},
		ModuleName: moduleName,
		FilePath:   moduleAbsPath,
		Location:   loc,
	}

	// Add dependency edge and check for cycles,
	p.ctx.AddDepEdge(p.filePathAbs, moduleAbsPath)

	// Always start cycle detection from the entrypoint
	entryRel := p.ctx.EntryPoint
	if p.ctx.RootDir != "" {
		entryRel = filepath.ToSlash(filepath.Join("", p.ctx.EntryPoint))
	}
	entryKey := entryRel
	if cycle, found := p.ctx.DetectCycle(entryKey); found {
		cycleStr := strings.Join(cycle, " -> ")
		msg := "Circular import detected: " + cycleStr
		p.ctx.Reports.Add(p.filePathAbs, &loc, msg, report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
		colors.RED.Println(msg)
		return stmt
	}

	// Check if the module is already cached
	if !p.ctx.HasModule(canonicalName) {

		module := NewParser(moduleAbsPath, p.ctx, p.debug).Parse()

		if module == nil {
			p.ctx.Reports.Add(p.filePathAbs, &loc, "Failed to parse imported module", report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
			return &ast.ImportStmt{Location: loc}
		}

		p.ctx.AddModule(canonicalName, module)
		colors.GREEN.Printf("Cached <- Module '%s'\n", canonicalName)
	} else {
		colors.ORANGE.Printf("Skipping module '%s' : Already cached\n", canonicalName)
	}

	p.ctx.AliasToModuleName[moduleName] = canonicalName

	fmt.Printf("Parsing import: %s -> %s\n", p.ctx.AbsToModuleName(p.filePathAbs), canonicalName)

	return stmt
}

func parseScopeResolution(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	// Handle scope resolution operator
	if module, ok := expr.(*ast.IdentifierExpr); ok {
		p.consume(lexer.SCOPE_TOKEN, report.EXPECTED_SCOPE_RESOLUTION_OPERATOR)
		if !p.match(lexer.IDENTIFIER_TOKEN) {
			token := p.peek()
			p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&token.Start, &token.End), "Expected identifier after '::'", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
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
		p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&token.Start, &token.End), "Left side of '::' must be an identifier", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}
}
