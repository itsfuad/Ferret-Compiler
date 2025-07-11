package parser

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"compiler/internal/utils/fs"
	"fmt"
	"strings"
)

// parseImport parses an import statement
func parseImport(p *Parser) ast.Node {

	start := p.consume(lexer.IMPORT_TOKEN, report.EXPECTED_IMPORT_KEYWORD)
	importToken := p.consume(lexer.STRING_TOKEN, report.EXPECTED_IMPORT_PATH)

	importpath := importToken.Value

	fmt.Printf("Parsing import statement in %s -> %s\n", p.fullPath, importpath)
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
			p.ctx.Reports.Add(p.fullPath, source.NewLocation(&start.Start, &importToken.End), report.INVALID_IMPORT_PATH, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		sufs := strings.Split(parts[len(parts)-1], ".")
		suf := "." + sufs[len(sufs)-1]
		moduleName = strings.TrimSuffix(parts[len(parts)-1], suf)
	}

	loc := *source.NewLocation(&start.Start, &importToken.End)

	moduleFullPath, err := fs.ResolveModule(importpath, p.fullPath, p.ctx)
	if err != nil {
		p.ctx.Reports.Add(p.fullPath, &loc, err.Error(), report.PARSING_PHASE).SetLevel(report.CRITICAL_ERROR)
		colors.RED.Println("Error resolving module:", err)
		return nil
	}

	isRemote := fs.IsRemote(importpath)
	if isRemote {
		fmt.Printf("Remote import detected: %s\n", importpath)
	} else {
		fmt.Printf("Local import detected: %s\n", importpath)
	}

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

	// Check if this module is already being parsed (circular dependency)
	if cycle, found := p.ctx.GetCyclePath(moduleFullPath); found {
		// Convert full paths to module names for better readability
		moduleNames := make([]string, len(cycle))
		for i, path := range cycle {
			moduleNames[i] = p.ctx.FullPathToModuleName(path)
		}

		cycleStr := strings.Join(moduleNames, " -> ")
		cycleMsg := fmt.Sprintf("Circular import detected: %s", cycleStr)
		p.ctx.Reports.Add(p.fullPath, &loc, cycleMsg, report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
		colors.RED.Println(cycleMsg)
		return stmt
	}

	// Add dependency edge for tracking
	if p.ctx.DepGraph == nil {
		p.ctx.DepGraph = make(map[string][]string)
	}
	p.ctx.DepGraph[p.fullPath] = append(p.ctx.DepGraph[p.fullPath], moduleFullPath)

	// If no cycle detected, we can proceed
	colors.CYAN.Printf("Dependency edge added: %s -> %s\n", p.fullPath, moduleFullPath)

	// Check if the module is already cached
	if !p.ctx.HasModule(importpath) {
		module := NewParser(moduleFullPath, p.ctx, p.debug).Parse()
		if module == nil {
			p.ctx.Reports.Add(p.fullPath, &loc, "Failed to parse imported module", report.PARSING_PHASE).SetLevel(report.SEMANTIC_ERROR)
			return &ast.ImportStmt{Location: loc}
		}
	}

	p.modulenameToImportpath[moduleName] = importpath

	colors.YELLOW.Println("Parsing import")
	fmt.Printf("Full path: %s, Module name: %s, Import path: %s\n", moduleFullPath, moduleName, importpath)

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
