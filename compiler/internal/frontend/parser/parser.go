package parser

import (
	"fmt"
	"path/filepath"
	"slices"

	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"compiler/internal/utils/fs"
)

type Parser struct {
	tokens     []lexer.Token
	tokenNo    int
	fullPath   string
	importPath string
	modulename string // module name derived from full path
	ctx        *ctx.CompilerContext
	debug      bool // debug mode for additional logging
}

func NewParser(filePath string, ctxx *ctx.CompilerContext, debug bool) *Parser {

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

	//relative path to the file
	importPath := ctxx.FullPathToImportPath(filePath)
	modulename := ctxx.FullPathToModuleName(filePath)

	tokens := lexer.Tokenize(filePath, false)

	return &Parser{
		tokens:     tokens,
		tokenNo:    0,
		ctx:        ctxx,
		fullPath:   filePath,
		importPath: importPath,
		modulename: modulename,
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

// parseExpressionList parses a comma-separated list of expressions
func parseExpressionList(p *Parser, first ast.Expression) ast.ExpressionList {
	exprs := ast.ExpressionList{first}
	for p.match(lexer.COMMA_TOKEN) {
		p.advance() // consume comma
		next := parseExpression(p)
		if next == nil {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected expression after comma", report.PARSING_PHASE)
			break
		}
		exprs = append(exprs, next)
	}
	return exprs
}

// parseExpressionStatement parses an expression statement
func parseExpressionStatement(p *Parser, first ast.Expression) ast.Statement {
	exprs := parseExpressionList(p, first)

	// Check for assignment
	if p.match(lexer.EQUALS_TOKEN) {
		return parseAssignment(p, exprs...)
	}

	return &ast.ExpressionStmt{
		Expressions: &exprs,
		Location:    *source.NewLocation(first.Loc().Start, exprs[len(exprs)-1].Loc().End),
	}
}

// handleUnexpectedToken reports an error for unexpected token and advances
func handleUnexpectedToken(p *Parser) ast.Statement {
	token := p.peek()
	p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End),
		fmt.Sprintf(report.UNEXPECTED_TOKEN+" `%s`", token.Value), report.PARSING_PHASE)

	p.advance() // skip the invalid token

	return nil
}

// parseBlock parses a block of statements
func parseBlock(p *Parser) *ast.Block {
	start := p.consume(lexer.OPEN_CURLY, report.EXPECTED_OPEN_BRACE).Start

	nodes := make([]ast.Node, 0)

	for !p.isAtEnd() && p.peek().Kind != lexer.CLOSE_CURLY {
		node := parseNode(p)
		if node != nil {
			nodes = append(nodes, node)
		}
	}

	end := p.consume(lexer.CLOSE_CURLY, report.EXPECTED_CLOSE_BRACE).End

	return &ast.Block{
		Nodes:    nodes,
		Location: *source.NewLocation(&start, &end),
	}
}

// parseReturnStmt parses a return statement
func parseReturnStmt(p *Parser) ast.Statement {

	start := p.consume(lexer.RETURN_TOKEN, report.EXPECTED_RETURN_KEYWORD).Start
	end := start

	// Check if there's a value to return
	var value ast.Expression
	if !p.match(lexer.SEMICOLON_TOKEN) {
		value = parseExpression(p)
		if value == nil {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), report.INVALID_EXPRESSION, report.PARSING_PHASE).AddHint("Add an expression after the return keyword")
		} else {
			end = *value.Loc().End

			// Check if user is trying to return multiple values
			if p.match(lexer.COMMA_TOKEN) {
				comma := p.peek()
				p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&comma.Start, &comma.End), "Multiple return values are not supported", report.PARSING_PHASE).AddHint("Functions can only return a single value")
				// Skip remaining expressions to continue parsing
				for p.match(lexer.COMMA_TOKEN) {
					p.advance() // consume comma
					if expr := parseExpression(p); expr != nil {
						end = *expr.Loc().End
					}
				}
			}
		}
	}

	return &ast.ReturnStmt{
		Value:    &value,
		Location: *source.NewLocation(&start, &end),
	}
}

// parseNode parses a single statement or expression
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
		} else {
			fmt.Printf("Invalid expression: %+v\n", expr)
			// if the expression is invalid, report an error
			node = handleUnexpectedToken(p)
		}
	default:
		fmt.Printf("Invalid token: %+v\n", p.peek())
		node = handleUnexpectedToken(p)
	}

	// Handle statement termination and update locations
	if _, ok := node.(ast.Statement); ok {
		//if no semicolon, show error on the previous token
		if !p.match(lexer.SEMICOLON_TOKEN) {
			token := p.previous()
			loc := source.NewLocation(&token.Start, &token.End)
			loc.Start.Column += 1
			loc.End.Column += 1
			p.ctx.Reports.AddSyntaxError(p.fullPath, loc, report.EXPECTED_SEMICOLON+" after "+token.Value, report.PARSING_PHASE).AddHint("Add a semicolon to the end of the statement")
		}
		end := p.advance()
		node.Loc().End.Column = end.End.Column
		node.Loc().End.Line = end.End.Line
	}

	return node
}

// Parse is the entry point for parsing
func (p *Parser) Parse() *ast.Program {
	var nodes []ast.Node

	// Start tracking the entry point parsing
	p.ctx.StartParsing(p.fullPath)
	// Finish tracking the entry point parsing
	defer p.ctx.FinishParsing(p.fullPath)

	for !p.isAtEnd() {
		// Parse the statement
		node := parseNode(p)
		if node != nil {
			nodes = append(nodes, node)
		} else {
			handleUnexpectedToken(p)
			break
		}
	}

	if len(nodes) == 0 {
		return &ast.Program{}
	}

	if p.debug {
		colors.BLUE.Printf("Parsed '%s'\n", p.fullPath)
	}

	program := &ast.Program{
		Nodes:      nodes,
		FullPath:   p.fullPath,
		ImportPath: p.importPath,
		Modulename: p.modulename,
		Location:   *source.NewLocation(&p.tokens[0].Start, nodes[len(nodes)-1].Loc().End),
	}

	// Add the module to the context
	p.ctx.AddModule(p.importPath, program)

	//show the ast
	if p.debug {
		program.SaveAST()
	}

	return program
}
