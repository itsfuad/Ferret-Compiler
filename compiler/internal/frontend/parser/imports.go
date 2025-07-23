package parser

import (
	"fmt"
	"strings"

	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"compiler/internal/utils/fs"
)

// parseImport parses an import statement
func parseImport(p *Parser) ast.Node {

	start := p.consume(lexer.IMPORT_TOKEN, report.EXPECTED_IMPORT_KEYWORD)
	importToken := p.consume(lexer.STRING_TOKEN, report.EXPECTED_IMPORT_PATH)

	importpath := importToken.Value

	// If we're parsing a remote module file and this import doesn't look like a remote import,
	// we need to convert it to a full remote import path
	if p.ctx.IsRemoteModuleFile(p.fullPath) && !fs.IsRemote(importpath) && !fs.IsBuiltinModule(fs.FirstPart(importpath)) {
		fullImportPath := p.convertLocalToRemoteImportPath(importpath)
		if fullImportPath != "" {
			importpath = fullImportPath
		}
	}

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
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&start.Start, &importToken.End), report.INVALID_IMPORT_PATH, report.PARSING_PHASE)
			return nil
		}
		sufs := strings.Split(parts[len(parts)-1], ".")
		suf := "." + sufs[len(sufs)-1]
		moduleName = strings.TrimSuffix(parts[len(parts)-1], suf)
	}

	loc := *source.NewLocation(&start.Start, &importToken.End)

	moduleFullPath, err := fs.ResolveModule(importpath, p.fullPath, p.ctx)
	if err != nil {
		p.ctx.Reports.AddCriticalError(p.fullPath, &loc, err.Error(), report.PARSING_PHASE)
		colors.RED.Println("Error resolving module:", err)
		return nil
	}

	stmt := &ast.ImportStmt{
		ImportPath: &ast.StringLiteral{
			Value:    importpath,
			Location: loc,
		},
		ModuleName: moduleName,
		FullPath:   moduleFullPath,
		Location:   loc,
	}

	// Check for circular dependency before adding the import
	if cycle, found := p.ctx.DetectCycle(p.fullPath, moduleFullPath); found {
		// Convert full paths to module names for better readability
		moduleNames := make([]string, len(cycle))
		for i, path := range cycle {
			moduleNames[i] = p.ctx.FullPathToImportPath(path)
		}

		cycleStr := strings.Join(moduleNames, " → ")
		currentModule := p.ctx.FullPathToImportPath(p.fullPath)
		targetModule := p.ctx.FullPathToImportPath(moduleFullPath)

		cycleMsg := fmt.Sprintf("Import cycle detected: %s\nProblem: %s cannot import %s (already in dependency path)",
			cycleStr, currentModule, targetModule)
		p.ctx.Reports.AddCriticalError(p.fullPath, &loc, cycleMsg, report.PARSING_PHASE)
		return stmt
	}

	// Check if the module is already parsed
	if !p.ctx.IsModuleParsed(importpath) {
		module := NewParser(moduleFullPath, p.ctx, p.debug).Parse()
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
	} else {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Left side of '::' must be an identifier", report.PARSING_PHASE)
		return nil, false
	}
}

// convertLocalToRemoteImportPath converts a local import path to a full remote import path
// when parsing a file that's inside a remote module cache
func (p *Parser) convertLocalToRemoteImportPath(localImportPath string) string {
	// We need to figure out which remote repo this file belongs to and
	// prepend the repo path to the local import

	// Get the remote import path of the current file
	currentRemoteImportPath := p.ctx.GetRemoteModuleImportPath(p.fullPath)
	if currentRemoteImportPath == "" {
		return ""
	}

	// Parse the current remote import to get repo and version info
	repoPath, version, _ := p.ctx.ParseRemoteImport(currentRemoteImportPath)
	if repoPath == "" {
		return ""
	}

	// Construct the full remote import path
	// e.g., "data/mods/constants" becomes "github.com/user/repo/data/mods/constants"
	if version == "latest" {
		return repoPath + "/" + localImportPath
	} else {
		return repoPath + "@" + version + "/" + localImportPath
	}
}
