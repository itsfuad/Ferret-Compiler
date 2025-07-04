package parser

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/frontend/lexer"
	"ferret/compiler/internal/source"
	"ferret/compiler/report"
	"strings"
)

// parseModule parses a package declaration
func parseModule(p *Parser) ast.Node {
	// Check if this is the first statement in the file

	valid := p.tokenNo == 0

	start := p.consume(lexer.MODULE_TOKEN, report.EXPECTED_PACKAGE_KEYWORD)

	name := p.consume(lexer.IDENTIFIER_TOKEN, report.EXPECTED_PACKAGE_NAME)

	if !valid {
		p.Reports.Add(p.filePath, source.NewLocation(&start.Start, &name.End), report.INVALID_SCOPE).AddHint("Package declarations must be at the top level of the file").SetLevel(report.SYNTAX_ERROR)
	}

	return &ast.ModuleDeclStmt{
		ModuleName: &ast.IdentifierExpr{
			Name:     name.Value,
			Location: *source.NewLocation(&name.Start, &name.End),
		},
		Location: *source.NewLocation(&start.Start, &name.End),
	}
}

// parseImport parses an import statement
func parseImport(p *Parser) ast.Node {
	start := p.consume(lexer.IMPORT_TOKEN, report.EXPECTED_IMPORT_KEYWORD)
	importPath := p.consume(lexer.STRING_TOKEN, report.EXPECTED_IMPORT_PATH)

	// Get module name from import path (last component)
	moduleName := strings.Trim(importPath.Value, "\"")
	if lastSlash := strings.LastIndex(moduleName, "/"); lastSlash >= 0 {
		moduleName = moduleName[lastSlash+1:]
	}

	loc := *source.NewLocation(&start.Start, &importPath.End)

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
			p.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Expected identifier after '::'").SetLevel(report.SYNTAX_ERROR)
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
		p.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Left side of '::' must be an identifier").SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}
}
