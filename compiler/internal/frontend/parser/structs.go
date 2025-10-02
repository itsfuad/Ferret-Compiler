package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"compiler/report"
	"fmt"
)

// validateStructType validates the struct type and returns the type name
func validateStructType(p *Parser) (*ast.IdentifierExpr, bool) {
	if !p.match(lexer.IDENTIFIER_TOKEN, lexer.STRUCT_TOKEN) {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "expected identifier or 'struct' keyword", report.PARSING_PHASE)
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
		fieldName := p.consume(lexer.IDENTIFIER_TOKEN, "expected field name")
		if fieldNames[fieldName.Value] {
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&fieldName.Start, &fieldName.End), fmt.Sprintf("field %q already defined", fieldName.Value), report.PARSING_PHASE)
			return nil, false
		}

		fieldNames[fieldName.Value] = true

		p.consume(lexer.COLON_TOKEN, "expected ':' after field name")

		value := parseExpression(p)
		if value == nil {
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&fieldName.Start, &fieldName.End), fmt.Sprintf("expected value for field %q", fieldName.Value), report.PARSING_PHASE).AddHint("add an expression after the colon")
			return nil, false
		}

		fields = append(fields, ast.StructField{
			FieldIdentifier: &ast.IdentifierExpr{
				Name:     fieldName.Value,
				Location: *source.NewLocation(&fieldName.Start, &fieldName.End),
			},
			FieldValue: value,
			Location:   *source.NewLocation(&fieldName.Start, value.Loc().End),
		})

		if p.match(lexer.CLOSE_CURLY) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, "expected ',' or '}' after struct field")
			if p.match(lexer.CLOSE_CURLY) {
				p.ctx.Reports.AddWarning(p.fullPath, source.NewLocation(&comma.Start, &comma.End), "unnecessary trailing comma after last struct field", report.PARSING_PHASE).AddHint("remove the trailing comma")
				break
			}
		}
	}

	return fields, true
}

// parseStructLiteral parses a struct literal expression like Point{x: 10, y: 20}
func parseStructLiteral(p *Parser) ast.Expression {

	start := p.consume(lexer.AT_TOKEN, "expected '@' before struct literal").Start

	typeName, ok := validateStructType(p)
	if !ok {
		return nil
	}

	p.consume(lexer.OPEN_CURLY, "expected '{' after struct name")

	if p.peek().Kind == lexer.CLOSE_CURLY {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End),
			"struct is empty", report.PARSING_PHASE).AddHint("struct must have at least one field")
		return nil
	}

	fields, ok := parseStructFields(p)
	if !ok {
		return nil
	}

	end := p.consume(lexer.CLOSE_CURLY, "expected '}' after struct fields").End

	return &ast.StructLiteralExpr{
		StructName:  typeName,
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
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End),
			"expected field name after '.'", report.PARSING_PHASE)
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
