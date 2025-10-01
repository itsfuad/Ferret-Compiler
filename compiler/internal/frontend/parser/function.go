package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"compiler/internal/utils"
	"compiler/internal/utils/lists"
	"compiler/report"
	"fmt"
)

// detect if it's a function or a method
//
// function: fn NAME (PARAMS) {BODY} // named
//
// function: fn (PARAMS) {BODY} // anonymous
//
// method: fn (r Receiver) NAME (PARAMS) {BODY}
//
// method: fn (r Receiver, others...) NAME (PARAMS) {BODY} // invalid, but we can still parse it and report an error
func parseFunctionLike(p *Parser) ast.Node {

	start := p.peek() // the fn token

	var params []ast.Parameter

	if p.next().Kind == lexer.OPEN_PAREN {
		p.advance() // consume the fn token
		// either a method or anonymous function
		// fn (PARAMS) {BODY} // anonymous
		// fn (PARAMS) NAME (PARAMS) {BODY} // method
		params = parseParameters(p)
		// if identifier, it's a method
		if p.match(lexer.IDENTIFIER_TOKEN) {
			return parseMethodDeclaration(p, &start.Start, params)
		}
		// anonymous function
		return parseFunctionLiteral(p, &start.Start, false, params...)
	}
	// named function
	return parseFunctionDecl(p)
}

func parseParameters(p *Parser) []ast.Parameter {

	params := []ast.Parameter{}

	p.consume(lexer.OPEN_PAREN, "expected '(' before function parameters")

	for !p.match(lexer.CLOSE_PAREN) {

		isVariadic := false

		if p.match(lexer.THREE_DOT_TOKEN) {
			isVariadic = true
			p.advance()
		}

		identifier := p.consume(lexer.IDENTIFIER_TOKEN, "expected parameter name")

		location := *source.NewLocation(&identifier.Start, &identifier.End)

		paramName := &ast.IdentifierExpr{Name: identifier.Value, Location: location}

		p.consume(lexer.COLON_TOKEN, "expected ':' after parameter name")

		paramType, ok := parseType(p)
		if !ok {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "no parameter type provided", report.PARSING_PHASE).AddLabel("add a type after the colon")
			return nil
		}

		param := ast.Parameter{
			Identifier: paramName,
			Type:       paramType,
			IsVariadic: isVariadic,
		}

		//check if the parameter is already defined
		if lists.Has(params, param, func(p ast.Parameter, b ast.Parameter) bool {
			return p.Identifier.Name == b.Identifier.Name
		}) {
			p.ctx.Reports.AddSyntaxError(p.fullPath, &param.Identifier.Location, fmt.Sprintf("parameter %q already defined", paramName.Name), report.PARSING_PHASE).AddHint("remove the duplicate parameter or rename it")
			return nil
		}

		params = append(params, param)

		if p.match(lexer.CLOSE_PAREN) {
			break
		}

		if p.match(lexer.CLOSE_PAREN) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, "expected ',' or ')' after parameter")
			if p.match(lexer.CLOSE_PAREN) {
				p.ctx.Reports.AddWarning(p.fullPath, source.NewLocation(&comma.Start, &comma.End), "unnecessary trailing comma after the last parameter", report.PARSING_PHASE).AddHint("remove the trailing comma")
				break
			}
		}
	}

	p.consume(lexer.CLOSE_PAREN, "expected ')' after function parameters")

	return params
}

func parseReturnType(p *Parser) ast.DataType {
	p.advance()

	// Check if user is trying to use multiple return types (parentheses)
	if p.peek().Kind == lexer.OPEN_PAREN {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "multiple return types are not supported", report.PARSING_PHASE).AddHint("functions can only return a single type")
		// Skip the entire parentheses expression to continue parsing
		p.advance() // consume '('
		parenCount := 1
		for parenCount > 0 && !p.isAtEnd() {
			if p.peek().Kind == lexer.OPEN_PAREN {
				parenCount++
			} else if p.peek().Kind == lexer.CLOSE_PAREN {
				parenCount--
			}
			p.advance()
		}
		return nil
	}

	// Parse single return type
	returnType, ok := parseType(p)
	if !ok {
		token := p.previous()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "no return type provided", report.PARSING_PHASE).AddLabel("add a return type after the arrow")
		return nil
	}

	return returnType
}

func parseSignature(p *Parser, paramNotParsedYet bool, params ...ast.Parameter) ([]ast.Parameter, ast.DataType) {

	if len(params) == 0 && paramNotParsedYet {
		params = parseParameters(p)
	}

	// Parse return type if present
	if p.match(lexer.ARROW_TOKEN) {
		returnType := parseReturnType(p)
		return params, returnType
	}

	return params, nil
}

func parseFunctionLiteral(p *Parser, start *source.Position, paramNotParsedYet bool, params ...ast.Parameter) *ast.FunctionLiteral {

	params, returnType := parseSignature(p, paramNotParsedYet, params...)

	block := parseBlock(p)

	location := *source.NewLocation(start, block.Loc().End)

	return &ast.FunctionLiteral{
		ID:         utils.GenerateFunctionLiteralID(),
		Params:     params,
		ReturnType: returnType,
		Body:       block,
		Location:   location,
	}
}

func declareFunction(p *Parser) *ast.IdentifierExpr {

	var name *ast.IdentifierExpr

	if p.match(lexer.IDENTIFIER_TOKEN) {
		token := p.advance()
		location := *source.NewLocation(&token.Start, &token.End)
		name = &ast.IdentifierExpr{
			Name:     token.Value,
			Location: location,
		}
	}

	return name
}

func parseFunctionDecl(p *Parser) ast.BlockConstruct {

	// consume the function token
	start := p.advance()

	name := declareFunction(p)

	function := parseFunctionLiteral(p, &start.Start, true)

	function.ID = name.Name // Set the function ID to the declared name

	return &ast.FunctionDecl{
		Identifier: name,
		Function:   function,
		Location:   *source.NewLocation(&start.Start, function.Loc().End),
	}
}
