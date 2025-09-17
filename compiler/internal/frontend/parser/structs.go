package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"compiler/report"
)

// validateStructType validates the struct type and returns the type name
func validateStructType(p *Parser) (*ast.IdentifierExpr, bool) {
	if !p.match(lexer.IDENTIFIER_TOKEN, lexer.STRUCT_TOKEN) {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), report.EXPECTED_TYPE, report.PARSING_PHASE)
		return nil, false
	}

	token := p.advance()
	typeName := &ast.IdentifierExpr{
		Name:     token.Value,
		Location: *source.NewLocation(&token.Start, &token.End),
	}

	return typeName, true
}

// parseCompositeFields parses the fields of a composite literal
func parseCompositeFields(p *Parser) ([]ast.CompositeField, bool) {
	fields := make([]ast.CompositeField, 0)

	for !p.match(lexer.CLOSE_CURLY) {
		// Parse key expression
		key := parseExpression(p)
		if key == nil {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected key expression", report.PARSING_PHASE)
			return nil, false
		}

		// Expect separator : or =>
		var separator string
		if p.match(lexer.COLON_TOKEN) {
			p.advance() // consume the separator
			separator = ":"
		} else if p.match(lexer.FAT_ARROW_TOKEN) {
			p.advance() // consume the separator
			separator = "=>"
		} else {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected ':' or '=>' after key", report.PARSING_PHASE)
			return nil, false
		}

		// Parse value expression
		value := parseExpression(p)
		if value == nil {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected value expression", report.PARSING_PHASE)
			return nil, false
		}

		fields = append(fields, ast.CompositeField{
			Key:       &key,
			Value:     &value,
			Separator: separator,
			Location:  *source.NewLocation(key.Loc().Start, value.Loc().End),
		})

		if p.match(lexer.CLOSE_CURLY) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_CURLY)
			if p.match(lexer.CLOSE_CURLY) {
				p.ctx.Reports.AddWarning(p.fullPath, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED, report.PARSING_PHASE).AddHint("remove the trailing comma")
				break
			}
		}
	}

	return fields, true
}

// parseCompositeLiteral parses a composite literal expression like Point{x: 10} or Map{key => val}
func parseCompositeLiteral(p *Parser) ast.Expression {
	// Parse type name
	typeName, ok := validateStructType(p)
	if !ok {
		return nil
	}

	start := typeName.Location.Start

	p.consume(lexer.OPEN_CURLY, report.EXPECTED_OPEN_BRACE)

	if p.peek().Kind == lexer.CLOSE_CURLY {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Empty composite literals are not allowed", report.PARSING_PHASE)
		return nil
	}

	fields, ok := parseCompositeFields(p)
	if !ok {
		return nil
	}

	end := p.consume(lexer.CLOSE_CURLY, report.EXPECTED_CLOSE_BRACE).End

	return &ast.CompositeLiteralExpr{
		TypeName:    typeName,
		IsAnonymous: false,
		Fields:      fields,
		Location:    *source.NewLocation(start, &end),
	}
}

// parseAnonymousCompositeLiteral parses an anonymous composite literal like @struct{x: 10, y: 20}
func parseAnonymousCompositeLiteral(p *Parser) ast.Expression {
	// Consume the '@' token
	atToken := p.advance()
	start := atToken.Start

	// Expect 'struct' keyword
	if !p.match(lexer.STRUCT_TOKEN) {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected 'struct' after '@'", report.PARSING_PHASE)
		return nil
	}
	p.advance() // consume 'struct'

	p.consume(lexer.OPEN_CURLY, report.EXPECTED_OPEN_BRACE)

	if p.peek().Kind == lexer.CLOSE_CURLY {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Empty composite literals are not allowed", report.PARSING_PHASE)
		return nil
	}

	fields, ok := parseCompositeFields(p)
	if !ok {
		return nil
	}

	end := p.consume(lexer.CLOSE_CURLY, report.EXPECTED_CLOSE_BRACE).End

	return &ast.CompositeLiteralExpr{
		TypeName:    nil,
		IsAnonymous: true,
		Fields:      fields,
		Location:    *source.NewLocation(&start, &end),
	}
}

// parseFieldAccess parses a field access expression like struct.field
func parseFieldAccess(p *Parser, object ast.Expression) (ast.Expression, bool) {
	p.advance() // consume '.'

	// Parse field name
	if !p.match(lexer.IDENTIFIER_TOKEN) {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End),
			"Expected field name after '.'", report.PARSING_PHASE)
		return nil, false
	}

	fieldToken := p.advance()
	field := &ast.IdentifierExpr{
		Name:     fieldToken.Value,
		Location: *source.NewLocation(&fieldToken.Start, &fieldToken.End),
	}

	return &ast.FieldAccessExpr{
		Object:   &object,
		Field:    field,
		Location: *source.NewLocation(object.Loc().Start, &fieldToken.End),
	}, true
}
