package parser

import (
	"compiler/cmd/resolver"
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"strings"
)

// parseImport parses an import statement
func parseImport(p *Parser) ast.Node {
	start := p.consume(lexer.IMPORT_TOKEN, report.EXPECTED_IMPORT_KEYWORD)
	importPath := p.consume(lexer.STRING_TOKEN, report.EXPECTED_IMPORT_PATH)

	// Get module name from import path (last component) [e.g., "std/fmt/log.fer | .ferret]

	parts := strings.Split(importPath.Value, "/")
	if len(parts) == 0 {
		p.ctx.Reports.Add(p.filePath, source.NewLocation(&start.Start, &importPath.End), report.INVALID_IMPORT_PATH, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil
	}

	sufs := strings.Split(parts[len(parts)-1], ".")
	suf := "." + sufs[len(sufs)-1]

	moduleName := strings.TrimSuffix(parts[len(parts)-1], suf)

	loc := *source.NewLocation(&start.Start, &importPath.End)

	colors.BLUE.Printf("Import module name: '%s', path: '%s'\n", moduleName, importPath.Value)

	resolvedPath, err := resolver.ResolveModule(importPath.Value, p.filePath, p.ctx, false)
	if err != nil {
		p.ctx.Reports.Add(p.filePath, &loc, err.Error(), report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return nil
	}

	colors.YELLOW.Printf("Resolved import path: '%s'\n", resolvedPath)

	// Check if the module is already cached
	if !p.ctx.HasModule(resolvedPath) {

		module := NewParser(resolvedPath, p.ctx, p.debug).Parse()

		if module == nil {
			p.ctx.Reports.Add(p.filePath, &loc, "Failed to parse imported module", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil
		}

		p.ctx.AddModule(resolvedPath, module)
		colors.GREEN.Printf("Module '%s' added to cache\n", moduleName)
	} else {
		colors.GREEN.Printf("Module '%s' already cached\n", moduleName)
	}

	return &ast.ImportStmt{
		ImportPath: &ast.StringLiteral{
			Value:    importPath.Value,
			Location: loc,
		},
		ModuleName: moduleName,
		Location:   loc,
	}
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
