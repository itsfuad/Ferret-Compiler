package parser

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/frontend/lexer"
	"ferret/compiler/internal/source"
	"ferret/compiler/report"
	"fmt"
	// "os" // No longer needed after removing Fprintf(os.Stderr,...)
	"slices"
)

type Parser struct {
	tokens      []lexer.Token
	tokenNo     int
	filePath    string
	fileQueue   []string
	parsedFiles map[string]bool
}

func New(filePath string, debug bool) *Parser {
	tokens := lexer.Tokenize(filePath, debug)
	return &Parser{
		tokens:      tokens,
		tokenNo:     0,
		filePath:    filePath,
		fileQueue:   []string{filePath},
		parsedFiles: make(map[string]bool),
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

	err := report.Add(p.filePath, source.NewLocation(&current.Start, &current.End), message)
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
			report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Expected expression after comma").SetLevel(report.SYNTAX_ERROR)
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
	report.Add(p.filePath, source.NewLocation(&token.Start, &token.End),
		fmt.Sprintf(report.UNEXPECTED_TOKEN+" `%s`", token.Value)).SetLevel(report.SYNTAX_ERROR)

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
			report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), report.INVALID_EXPRESSION).AddHint("Add an expression after the return keyword").SetLevel(report.SYNTAX_ERROR)
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
	case lexer.MODULE_TOKEN:
		node = parseModule(p)
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
			prevToken := p.previous()
			// peekToken := p.peek() // Debug line, can be removed
			// fmt.Printf("DEBUG: Semicolon check failed. File: %s. Previous token: '%s' (Kind: %s). Current (Peek) token: '%s' (Kind: %s). Previous Start: %v, End: %v. Peek Start: %v, End: %v\n",
			// 	p.filePath, prevToken.Value, prevToken.Kind, peekToken.Value, peekToken.Kind, prevToken.Start, prevToken.End, peekToken.Start, peekToken.End)

			loc := source.NewLocation(&prevToken.Start, &prevToken.End)
			report.Add(p.filePath, loc, report.EXPECTED_SEMICOLON+" after "+prevToken.Value).AddHint("Add a semicolon to the end of the statement").SetLevel(report.SYNTAX_ERROR)
		}
		end := p.advance()
		node.Loc().End.Column = end.End.Column
		node.Loc().End.Line = end.End.Line
	}

	return node
}

// parseCurrentFile is the entry point for parsing
func (p *Parser) parseCurrentFile() *ast.Program {

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
		Location: *source.NewLocation(&p.tokens[0].Start, nodes[len(nodes)-1].Loc().End),
	}
}

// ParseProgram is the entry point for parsing a program that may span multiple files.
func (p *Parser) ParseProgram() *ast.MultiProgram {
	multiProgram := &ast.MultiProgram{Programs: make(map[string]*ast.Program)}
	processedInitialFile := false

	for len(p.fileQueue) > 0 {
		// Dequeue
		filePathToParse := p.fileQueue[0]
		p.fileQueue = p.fileQueue[1:]

		if _, alreadyParsed := p.parsedFiles[filePathToParse]; alreadyParsed {
			continue
		}

		// The lexer.Tokenize function handles file reading and reports I/O errors.
		// If Tokenize encounters an error, it reports it and returns empty tokens.
		// For the very first file (entry point), tokens are already loaded by New().
		// For subsequent files, we load them here.
		if !processedInitialFile && filePathToParse == p.filePath {
			// This is the initial file path that New() already tokenized.
			// Its tokens are in p.tokens. We don't need to re-tokenize.
			// However, p.filePath is already correctly set by New().
			processedInitialFile = true
		} else {
			// This is a new file from the queue, or we want to ensure fresh tokenization
			// if New() behavior changes.
			p.tokens = lexer.Tokenize(filePathToParse, false) // Assuming debug is false
			p.filePath = filePathToParse
			p.tokenNo = 0
		}

		// If tokenization failed (e.g., file not found), tokens might be empty or only EOF.
		// parseCurrentFile should handle this gracefully (e.g. return empty Program).
		// lexer.Tokenize itself uses the report package for file I/O errors.
		// The check for p.tokens[0].Start.File was removed as Token/Position doesn't store File.
		// Check if tokenization failed or yielded no useful tokens.
		// lexer.Tokenize itself reports detailed errors (like file not found or actual tokenization errors).
		if len(p.tokens) == 0 || (len(p.tokens) == 1 && p.tokens[0].Kind == lexer.EOF_TOKEN) {
			// Add a high-level error indicating this file could not be processed.
			// Create a placeholder location for file-level errors (e.g., start of file).
			// Assuming Position struct is {Line int, Column int, Offset int}
			// And NewLocation takes two *Position args.
			// A more specific file-level location might be introduced in 'source' package later.
			startPos := source.Position{Line: 1, Column: 0, Index: 0} // Line 1, Col 0, Index 0
			endPos := source.Position{Line: 1, Column: 0, Index: 0}   // Same for a zero-length location
			fileLocation := source.NewLocation(&startPos, &endPos)

			// Check if lexer.Tokenize itself reported any errors. If not, this is more of a "file empty or unreadable" case.
			// For now, we add a generic error. The `report` package might need more nuanced error querying.
			report.Add(filePathToParse, fileLocation, "Failed to read or tokenize imported file. Check file existence and content.").SetLevel(report.NORMAL_ERROR)

			p.parsedFiles[filePathToParse] = true // Mark as processed to avoid re-queueing
			continue                              // Skip parsing this file
		}

		program := p.parseCurrentFile()
		multiProgram.Programs[filePathToParse] = program
		p.parsedFiles[filePathToParse] = true
	}

	return multiProgram
}
