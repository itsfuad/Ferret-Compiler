package parser

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/frontend/lexer"
	"ferret/compiler/internal/source"
	"ferret/compiler/report"
	"path/filepath"
	"strings"
)

// parseModule parses a package declaration
func parseModule(p *Parser) ast.Node {
	// Check if this is the first statement in the file

	valid := p.tokenNo == 0

	start := p.consume(lexer.MODULE_TOKEN, report.EXPECTED_PACKAGE_KEYWORD)

	name := p.consume(lexer.IDENTIFIER_TOKEN, report.EXPECTED_PACKAGE_NAME)

	if !valid {
		report.Add(p.filePath, source.NewLocation(&start.Start, &name.End), report.INVALID_SCOPE).AddHint("Package declarations must be at the top level of the file").SetLevel(report.SYNTAX_ERROR)
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
	importPathToken := p.consume(lexer.STRING_TOKEN, report.EXPECTED_IMPORT_PATH)

	originalImportPath := strings.Trim(importPathToken.Value, "\"")

	// Resolve the import path to a file path
	// For now, assume ".ferret" extension and relative to current file's directory
	resolvedPath := originalImportPath
	if !strings.HasSuffix(resolvedPath, ".ferret") {
		resolvedPath += ".ferret"
	}

	currentDir := filepath.Dir(p.filePath)
	absResolvedPath, err := filepath.Abs(filepath.Join(currentDir, resolvedPath))
	if err != nil {
		// Error resolving path, report it or handle as appropriate
		// For now, we'll let it proceed, but a real compiler might halt or warn
		report.Add(p.filePath, source.NewLocation(&importPathToken.Start, &importPathToken.End), "Could not resolve import path: "+err.Error()).SetLevel(report.WARNING)
	} else {
		resolvedPath = absResolvedPath
		// Check if the file has already been parsed or is in the queue
		_, alreadyParsed := p.parsedFiles[resolvedPath]
		inQueue := false
		for _, qPath := range p.fileQueue {
			if qPath == resolvedPath {
				inQueue = true
				break
			}
		}

		if !alreadyParsed && !inQueue {
			p.fileQueue = append(p.fileQueue, resolvedPath)
		}
	}

	// Get module name from original import path (last component)
	moduleName := originalImportPath
	if lastSlash := strings.LastIndex(moduleName, "/"); lastSlash >= 0 {
		moduleName = moduleName[lastSlash+1:]
	}
	if ext := filepath.Ext(moduleName); ext == ".ferret" {
		moduleName = strings.TrimSuffix(moduleName, ext)
	}


	loc := *source.NewLocation(&start.Start, &importPathToken.End)

	return &ast.ImportStmt{
		ImportPath: &ast.StringLiteral{
			Value:    importPathToken.Value, // Store the original string literal
			Location: loc,
		},
		ModuleName: moduleName, // This might need to be the normalized name
		Location:   loc,
	}
}

func parseScopeResolution(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	// Handle scope resolution operator
	if module, ok := expr.(*ast.IdentifierExpr); ok {
		p.consume(lexer.SCOPE_TOKEN, report.EXPECTED_SCOPE_RESOLUTION_OPERATOR)
		if !p.match(lexer.IDENTIFIER_TOKEN) {
			token := p.peek()
			report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Expected identifier after '::'").SetLevel(report.SYNTAX_ERROR)
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
		report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Left side of '::' must be an identifier").SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}
}
