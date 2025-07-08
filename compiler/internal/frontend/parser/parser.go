package parser

import (
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"fmt"
	"slices"
)

type Parser struct {
	tokens      []lexer.Token
	tokenNo     int
	filePathAbs string
	ctx         *ctx.CompilerContext
	debug       bool // debug mode for additional logging
}

func NewParser(filePath string, ctx *ctx.CompilerContext, debug bool) *Parser {

	tokens := lexer.Tokenize(filePath, false)
	return &Parser{
		tokens:      tokens,
		tokenNo:     0,
		ctx:         ctx,
		filePathAbs: filePath,
		debug:       debug,
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

	err := p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&current.Start, &current.End), message, report.PARSING_PHASE)
	err.SetLevel(report.SYNTAX_ERROR)
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
			p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&token.Start, &token.End), "Expected expression after comma", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
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
	p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&token.Start, &token.End),
		fmt.Sprintf(report.UNEXPECTED_TOKEN+" `%s`", token.Value), report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)

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
	// Check if there's a values to return
	var values ast.ExpressionList
	if !p.match(lexer.SEMICOLON_TOKEN) {
		values = parseExpressionList(p, parseExpression(p))
		if values == nil {
			token := p.peek()
			p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&token.Start, &token.End), report.INVALID_EXPRESSION, report.PARSING_PHASE).AddHint("Add an expression after the return keyword").SetLevel(report.SYNTAX_ERROR)
		}
		end = *values.Loc().End
	}

	return &ast.ReturnStmt{
		Values:   values,
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
			p.ctx.Reports.Add(p.filePathAbs, loc, report.EXPECTED_SEMICOLON+" after "+token.Value, report.PARSING_PHASE).AddHint("Add a semicolon to the end of the statement").SetLevel(report.SYNTAX_ERROR)
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

	return &ast.Program{
		Nodes:    nodes,
		FilePath: p.filePathAbs,
		Location: *source.NewLocation(&p.tokens[0].Start, nodes[len(nodes)-1].Loc().End),
	}
}
