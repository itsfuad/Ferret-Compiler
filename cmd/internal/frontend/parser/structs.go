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
		p.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End), report.EXPECTED_TYPE_NAME).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}

	token := p.advance()
	typeName := &ast.IdentifierExpr{
		Name:     token.Value,
		Location: *source.NewLocation(&token.Start, &token.End),
	}

	return typeName, true
}

// parseStructFields parses the fields of a struct literal
func parseStructFields(p *Parser) ([]ast.StructField, bool) {
	fieldNames := make(map[string]bool)
	fields := make([]ast.StructField, 0)

	for !p.match(lexer.CLOSE_CURLY) {
		fieldName := p.consume(lexer.IDENTIFIER_TOKEN, report.EXPECTED_FIELD_NAME)
		if fieldNames[fieldName.Value] {
			p.Reports.Add(p.filePath, source.NewLocation(&fieldName.Start, &fieldName.End), report.DUPLICATE_FIELD_NAME).SetLevel(report.SYNTAX_ERROR)
			return nil, false
		}
		fieldNames[fieldName.Value] = true
		p.consume(lexer.COLON_TOKEN, report.EXPECTED_COLON)

		value := parseExpression(p)
		if value == nil {
			p.Reports.Add(p.filePath, source.NewLocation(&fieldName.Start, &fieldName.End), report.EXPECTED_FIELD_VALUE).AddHint("Add an expression after the colon").SetLevel(report.SYNTAX_ERROR)
			return nil, false
		}

		fields = append(fields, ast.StructField{
			FieldIdentifier: ast.IdentifierExpr{
				Name:     fieldName.Value,
				Location: *source.NewLocation(&fieldName.Start, &fieldName.End),
			},
			FieldValue: value,
			Location:   *source.NewLocation(&fieldName.Start, value.Loc().End),
		})

		if p.match(lexer.CLOSE_CURLY) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_CURLY)
			if p.match(lexer.CLOSE_CURLY) {
				p.Reports.Add(p.filePath, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED).AddHint("Remove the trailing comma").SetLevel(report.WARNING)
				break
			}
		}
	}

	return fields, true
}

// parseStructLiteral parses a struct literal expression like Point{x: 10, y: 20}
func parseStructLiteral(p *Parser) ast.Expression {
	start := p.consume(lexer.AT_TOKEN, report.EXPECTED_AT_TOKEN).Start

	typeName, ok := validateStructType(p)
	if !ok {
		return nil
	}

	p.consume(lexer.OPEN_CURLY, report.EXPECTED_OPEN_BRACE)

	if p.peek().Kind == lexer.CLOSE_CURLY {
		token := p.peek()
		p.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End),
			report.EMPTY_STRUCT_NOT_ALLOWED).SetLevel(report.SYNTAX_ERROR)
		return nil
	}

	fields, ok := parseStructFields(p)
	if !ok {
		return nil
	}

	end := p.consume(lexer.CLOSE_CURLY, report.EXPECTED_CLOSE_BRACE).End

	return &ast.StructLiteralExpr{
		StructName:  *typeName,
		Fields:      fields,
		IsAnonymous: lexer.TOKEN(typeName.Name) == lexer.STRUCT_TOKEN,
		Location:    *source.NewLocation(&start, &end),
	}
}

// parseFieldAccess parses a field access expression like struct.field
func parseFieldAccess(p *Parser, object ast.Expression) (ast.Expression, bool) {
	p.advance() // consume '.'

	// Parse field name
	if !p.match(lexer.IDENTIFIER_TOKEN) {
		token := p.peek()
		p.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End),
			"Expected field name after '.'").SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}

	fieldToken := p.advance()
	field := &ast.IdentifierExpr{
		Name:     fieldToken.Value,
		Location: *source.NewLocation(&fieldToken.Start, &fieldToken.End),
	}

	return &ast.FieldAccessExpr{
		Object:   object,
		Field:    field,
		Location: *source.NewLocation(object.Loc().Start, &fieldToken.End),
	}, true
}
