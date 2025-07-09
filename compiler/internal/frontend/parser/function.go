package parser

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"compiler/internal/utils/lists"
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

	start := p.peek()

	var params []ast.Parameter

	if p.next().Kind == lexer.OPEN_PAREN {
		p.advance()
		// either a method or anonymous function
		// fn (PARAMS) {BODY} // anonymous
		// fn (PARAMS) NAME (PARAMS) {BODY} // method
		params = parseParameters(p)
		// if identifier, it's a method
		if p.match(lexer.IDENTIFIER_TOKEN) {
			return parseMethodDeclaration(p, &start.Start, params)
		}
		// anonymous function
		return parseFunctionLiteral(p, &start.Start, true, false, params...)
	} else {
		// named function
		return parseFunctionDecl(p)
	}
}

func parseParameters(p *Parser) []ast.Parameter {

	params := []ast.Parameter{}

	p.consume(lexer.OPEN_PAREN, report.EXPECTED_OPEN_PAREN)

	for !p.match(lexer.CLOSE_PAREN) {

		identifier := p.consume(lexer.IDENTIFIER_TOKEN, report.EXPECTED_PARAMETER_NAME)

		location := *source.NewLocation(&identifier.Start, &identifier.End)

		paramName := &ast.IdentifierExpr{Name: identifier.Value, Location: location}

		p.consume(lexer.COLON_TOKEN, report.EXPECTED_COLON)

		paramType, ok := parseType(p)
		if !ok {
			token := p.peek()
			p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&token.Start, &token.End), report.EXPECTED_PARAMETER_TYPE, report.PARSING_PHASE).AddHint("Add a type after the colon").SetLevel(report.SYNTAX_ERROR)
			return nil
		}

		param := ast.Parameter{
			Identifier: paramName,
			Type:       paramType,
		}

		//check if the parameter is already defined
		if lists.Has(params, param, func(p ast.Parameter, b ast.Parameter) bool {
			return p.Identifier.Name == b.Identifier.Name
		}) {
			p.ctx.Reports.Add(p.filePathAbs, &param.Identifier.Location, report.PARAMETER_REDEFINITION, report.PARSING_PHASE).AddHint("Parameter name already used").SetLevel(report.SEMANTIC_ERROR)
			return nil
		}

		params = append(params, param)

		if p.match(lexer.CLOSE_PAREN) {
			break
		}

		if p.match(lexer.CLOSE_PAREN) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_PAREN)
			if p.match(lexer.CLOSE_PAREN) {
				p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED, report.PARSING_PHASE).AddHint("Remove the trailing comma").SetLevel(report.WARNING)
				break
			}
		}
	}

	p.consume(lexer.CLOSE_PAREN, report.EXPECTED_CLOSE_PAREN)

	return params
}

func parseReturnTypes(p *Parser) []ast.DataType {
	p.advance()
	// Check for multiple return types in parentheses
	if p.peek().Kind != lexer.OPEN_PAREN {
		// Single return type
		returnType, ok := parseType(p)
		if !ok {
			token := p.previous()
			p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&token.Start, &token.End), report.EXPECTED_RETURN_TYPE, report.PARSING_PHASE).AddHint("Add a return type after the arrow").SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		return []ast.DataType{returnType}
	}

	p.advance() // consume '('
	returnTypes := make([]ast.DataType, 0)

	for !p.match(lexer.CLOSE_PAREN) {
		returnType, ok := parseType(p)
		if !ok {
			token := p.previous()
			p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&token.Start, &token.End), report.EXPECTED_RETURN_TYPE, report.PARSING_PHASE).AddHint("Add a return type after the arrow").SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		returnTypes = append(returnTypes, returnType)

		if p.match(lexer.CLOSE_PAREN) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_PAREN)
			if p.match(lexer.CLOSE_PAREN) {
				p.ctx.Reports.Add(p.filePathAbs, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED, report.PARSING_PHASE).AddHint("Remove the trailing comma").SetLevel(report.WARNING)
				break
			}
		}
	}

	p.consume(lexer.CLOSE_PAREN, report.EXPECTED_CLOSE_PAREN)
	return returnTypes
}

func parseSignature(p *Parser, parseNewParams bool, params ...ast.Parameter) ([]ast.Parameter, []ast.DataType) {

	if len(params) == 0 && parseNewParams {
		params = parseParameters(p)
	}

	// Parse return type if present
	if p.match(lexer.ARROW_TOKEN) {
		returnTypes := parseReturnTypes(p)
		return params, returnTypes
	}

	return params, nil
}

func parseFunctionLiteral(p *Parser, start *source.Position, isAnonymous, parseNewParams bool, params ...ast.Parameter) *ast.FunctionLiteral {

	params, returnTypes := parseSignature(p, parseNewParams, params...)

	block := parseBlock(p)

	location := *source.NewLocation(start, block.Loc().End)

	return &ast.FunctionLiteral{
		Params:     params,
		ReturnType: returnTypes,
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

	colors.BLUE.Println("Parsing function declaration")

	// consume the function token
	start := p.consume(lexer.FUNCTION_TOKEN, report.EXPECTED_FUNCTION_KEYWORD)

	name := declareFunction(p)

	function := parseFunctionLiteral(p, &start.Start, false, true)

	return &ast.FunctionDecl{
		Identifier: *name,
		Function:   function,
		Location:   *source.NewLocation(&start.Start, function.Loc().End),
	}
}
