package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"compiler/colors"
	"compiler/config"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"compiler/internal/utils/fs"
	"compiler/report"
)

type Parser struct {
	tokens     []lexer.Token
	tokenNo    int
	fullPath   string
	importPath string
	alias      string
	ctx        *ctx.CompilerContext
	debug      bool // debug mode for additional logging
}

func NewParser(filePath string, ctxx *ctx.CompilerContext, debug bool) *Parser {

	rel, err := filepath.Rel(ctxx.ProjectRootFullPath, filePath)
	if err != nil {
		return nil
	}

	rel = strings.TrimSuffix(rel, filepath.Ext(rel))
	rel = filepath.Join(ctxx.ProjectConfig.Name, rel) // Prepend project name
	importPath := filepath.ToSlash(rel)

	return NewParserWithImportPath(filePath, importPath, ctxx, debug)
}

func NewParserWithImportPath(filePath string, importPath string, ctxx *ctx.CompilerContext, debug bool) *Parser {

	if ctxx == nil {
		panic("Cannot create parser: Compiler context is nil")
	}
	if filePath == "" {
		panic("Cannot create parser: File path is empty")
	}

	filePath = filepath.ToSlash(filePath) // Ensure forward slashes for consistency

	if !fs.IsValidFile(filePath) {
		panic(fmt.Sprintf("Cannot create parser: Invalid file path: %s", filePath))
	}

	alias := ctxx.FullPathToAlias(filePath)

	tokens := lexer.Tokenize(filePath, debug)

	return &Parser{
		tokens:     tokens,
		tokenNo:    0,
		fullPath:   filePath,
		importPath: importPath,
		alias:      alias,
		ctx:        ctxx,
		debug:      debug,
	}
}

// current token
func (p *Parser) peek() lexer.Token {
	return p.tokens[p.tokenNo]
}

// previous token
func (p *Parser) previous() lexer.Token {
	return p.tokens[p.tokenNo-1]
}

// next returns the next token without consuming it
func (p *Parser) next() lexer.Token {
	if p.tokenNo+1 >= len(p.tokens) {
		return lexer.Token{Kind: lexer.EOF_TOKEN}
	}
	return p.tokens[p.tokenNo+1]
}

// is at end of file
func (p *Parser) isAtEnd() bool {
	return p.peek().Kind == lexer.EOF_TOKEN
}

// consume the current token and return that token
func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.tokenNo++
	}
	return p.previous()
}

// check if the current token is of the given kind
func (p *Parser) check(kind lexer.TOKEN) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Kind == kind
}

// matches the current token with any of the given kinds
func (p *Parser) match(kinds ...lexer.TOKEN) bool {
	if p.isAtEnd() {
		return false
	}

	return slices.Contains(kinds, p.peek().Kind)
}

// consume the current token if it is of the given kind and return that token
// otherwise, report an error
func (p *Parser) consume(kind lexer.TOKEN, message string) lexer.Token {
	if p.check(kind) {
		return p.advance()
	}

	current := p.peek()

	p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&current.Start, &current.End), message, report.PARSING_PHASE)

	return p.peek()
}

// handleUnexpectedToken reports an error for unexpected token and advances
func handleUnexpectedToken(p *Parser, expected string) ast.Statement {
	token := p.peek()
	p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End),
		fmt.Sprintf("expected %s, found unexpected token `%s`", expected, token.Value), report.PARSING_PHASE)

	p.advance() // skip the invalid token

	return nil
}

// parseNode parses a single statement or expression based on the current token.
// This is the main dispatch function that handles all Ferret language constructs.
//
// The parsing follows a recursive descent approach where:
// 1. Keywords (import, let, const, type, return, function, if) are handled by specific parsers
// 2. '@' token indicates struct literals
// 3. Identifiers are parsed as expressions, potentially becoming expression statements
// 4. Unknown tokens result in syntax errors
//
// Special handling:
// - Function literals can be immediately invoked (IIFE pattern)
// - Statements require semicolon termination
// - Location information is updated for proper error reporting
func parseNode(p *Parser) ast.Node {
	var node ast.Node
	switch p.peek().Kind {
	case lexer.IMPORT_TOKEN:
		node = parseImport(p)
	case lexer.LET_TOKEN, lexer.CONST_TOKEN:
		node = parseVarDecl(p)
	case lexer.TYPE_TOKEN:
		node = parseTypeDecl(p)
	case lexer.RETURN_TOKEN:
		node = parseReturnStmt(p)
	case lexer.FUNCTION_TOKEN:
		node = parseFunctionLike(p)
		// Check if this is a function literal that should be treated as an expression (IIFE)
		if funcLit, ok := node.(*ast.FunctionLiteral); ok {
			// If followed by '(', treat as an expression statement for IIFE
			if p.peek().Kind == lexer.OPEN_PAREN {
				// Create a function call expression with the literal as the callee
				callExpr, _ := parseFunctionCall(p, funcLit)
				node = parseExpressionStatement(p, callExpr)
			}
			// Otherwise, it's a standalone function literal at top level (unusual but valid)
		}
	case lexer.IF_TOKEN:
		node = parseIfStatement(p)
	case lexer.AT_TOKEN:
		node = parseStructLiteral(p)
	case lexer.IDENTIFIER_TOKEN:
		// Look ahead to see if this is an assignment
		expr := parseExpression(p)
		if expr != nil {
			// if the expression is valid, parse it as an expression statement
			node = parseExpressionStatement(p, expr)
		}
	}

	if node == nil {
		node = handleUnexpectedToken(p, "statement")
	}

	// Handle statement termination and update locations
	if _, ok := node.(ast.Statement); ok {
		//if no semicolon, show error on the previous token
		if !p.match(lexer.SEMICOLON_TOKEN) {
			token := p.previous()
			loc := source.NewLocation(&token.Start, &token.End)
			loc.Start.Column += 1
			loc.End.Column += 1
			p.ctx.Reports.AddSyntaxError(p.fullPath, loc, report.EXPECTED_SEMICOLON+" after "+token.Value, report.PARSING_PHASE).AddHint("add a semicolon to the end of the statement")
		}
		end := p.advance()
		node.Loc().End.Column = end.End.Column
		node.Loc().End.Line = end.End.Line
	}

	return node
}

// Parse is the main entry point for parsing a Ferret source file.
// It orchestrates the entire parsing process and returns a complete AST.
//
// The parsing process:
// 1. Initializes project configuration and context
// 2. Iteratively parses top-level statements until EOF
// 3. Handles syntax errors gracefully by reporting and continuing
// 4. Constructs the final Program AST with proper location information
// 5. Registers the parsed module with the compiler context
//
// Error handling:
// - Project configuration errors cause immediate termination
// - Syntax errors are reported but don't stop parsing
// - Empty files result in empty programs
//
// Returns:
// - *ast.Program: Complete abstract syntax tree for the file
// - Panics if module alias cannot be determined (indicates file path issues)
func (p *Parser) Parse() *ast.Program {
	var nodes []ast.Node

	projectRoot, err := config.GetProjectRoot(p.fullPath)
	if err != nil {
		colors.RED.Println("❌ Error getting project root:", err)
		os.Exit(1)
	}

	config, _ := config.LoadProjectConfig(projectRoot)

	p.ctx.ProjectStack.Push(config)
	//p.ctx.MarkParseStart(p.fullPath)
	//defer p.ctx.MarkParseFinish(p.fullPath)
	defer p.ctx.ProjectStack.Pop()

	for !p.isAtEnd() {
		// Parse the statement
		node := parseNode(p)
		if node != nil {
			nodes = append(nodes, node)
			continue
		}

		handleUnexpectedToken(p, "statement")
		break
	}

	if len(nodes) == 0 {
		return &ast.Program{}
	}

	if p.debug {
		colors.BLUE.Printf("Parsed %q\n", p.fullPath)
	}

	if p.alias == "" {
		panic("Module name cannot be empty, please check the file path: " + p.fullPath)
	}

	program := &ast.Program{
		Nodes:      nodes,
		FullPath:   p.fullPath,
		ImportPath: p.importPath,
		Alias:      p.alias,
		Location:   *source.NewLocation(&p.tokens[0].Start, nodes[len(nodes)-1].Loc().End),
	}

	p.ctx.AddModule(p.importPath, program)

	return program
}
