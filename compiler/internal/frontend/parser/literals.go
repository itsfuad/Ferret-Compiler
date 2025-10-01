package parser

import (
	"strings"

	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"compiler/internal/utils/numeric"
	"compiler/report"
)

func parseNumberLiteral(p *Parser) ast.Expression {
	number := p.consume(lexer.NUMBER_TOKEN, "expected number literal")
	raw := number.Value
	value := strings.ReplaceAll(raw, "_", "") // Remove underscores
	loc := *source.NewLocation(&number.Start, &number.End)

	// Try parsing as integer first
	if numeric.IsHexadecimal(value) {
		intVal, err := numeric.StringToInteger(value)
		if err != nil {
			p.ctx.Reports.AddSyntaxError(p.fullPath, &loc, "invalid hexadecimal literal", report.PARSING_PHASE)
		}

		return &ast.IntLiteral{
			Value:    intVal,
			Raw:      raw,
			Base:     16,
			Location: loc,
		}
	}

	if numeric.IsOctal(value) {
		intVal, err := numeric.StringToInteger(value)
		if err != nil {
			p.ctx.Reports.AddSyntaxError(p.fullPath, &loc, "invalid octal literal", report.PARSING_PHASE)
			return nil
		}
		return &ast.IntLiteral{
			Value:    intVal,
			Raw:      raw,
			Base:     8,
			Location: loc,
		}
	}

	if numeric.IsBinary(value) {
		intVal, err := numeric.StringToInteger(value)
		if err != nil {
			p.ctx.Reports.AddSyntaxError(p.fullPath, &loc, "invalid binary literal", report.PARSING_PHASE)
			return nil
		}
		return &ast.IntLiteral{
			Value:    intVal,
			Raw:      raw,
			Base:     2,
			Location: loc,
		}
	}

	// Try as decimal integer
	if numeric.IsDecimal(value) {
		intVal, err := numeric.StringToInteger(value)
		if err != nil {
			p.ctx.Reports.AddSyntaxError(p.fullPath, &loc, "invalid integer literal", report.PARSING_PHASE)
			return nil
		}
		return &ast.IntLiteral{
			Value:    intVal,
			Raw:      raw,
			Base:     10,
			Location: loc,
		}
	}

	// Then try as float (including scientific notation)
	if numeric.IsFloat(value) {
		floatVal, err := numeric.StringToFloat(value)
		if err != nil {
			p.ctx.Reports.AddSyntaxError(p.fullPath, &loc, "invalid float literal", report.PARSING_PHASE)
			return nil
		}

		return &ast.FloatLiteral{
			Value:    floatVal,
			Raw:      raw,
			Location: loc,
		}
	}

	// If neither, it's an invalid number format
	p.ctx.Reports.AddSyntaxError(p.fullPath, &loc, "invalid nummeric literal", report.PARSING_PHASE)
	return nil
}

func parseStringLiteral(p *Parser) ast.Expression {
	stringLiteral := p.consume(lexer.STRING_TOKEN, "expected string literal")
	loc := *source.NewLocation(&stringLiteral.Start, &stringLiteral.End)

	return &ast.StringLiteral{
		Value:    stringLiteral.Value,
		Location: loc,
	}
}

func parseByteLiteral(p *Parser) ast.Expression {
	byteLiteral := p.consume(lexer.BYTE_TOKEN, "expected byte literal")
	loc := *source.NewLocation(&byteLiteral.Start, &byteLiteral.End)

	return &ast.ByteLiteral{
		Value:    byteLiteral.Value,
		Location: loc,
	}
}

func parseArrayLiteral(p *Parser) ast.Expression {
	start := p.advance().Start // consume '['
	elements := make([]ast.Expression, 0)

	for !p.isAtEnd() && !p.match(lexer.CLOSE_BRACKET) {
		expr := parseExpression(p)
		if expr != nil {
			elements = append(elements, expr)
		}

		if p.match(lexer.CLOSE_BRACKET) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, "expected ',' or ']' after array element")
			if p.match(lexer.CLOSE_BRACKET) {
				p.ctx.Reports.AddWarning(p.fullPath, source.NewLocation(&comma.Start, &comma.End), "unnecessary trailing comma after last array element", report.PARSING_PHASE).AddHint("remove the trailing comma")
				break
			}
		}
	}

	end := p.consume(lexer.CLOSE_BRACKET, "expected ']' after array literal")

	// at least one element required
	if len(elements) == 0 {
		peek := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&peek.Start, &peek.End), "empty array", report.PARSING_PHASE).AddHint("array literals must contain at least one element to infer the type")
		return nil
	}

	return &ast.ArrayLiteralExpr{
		Elements: elements,
		Location: *source.NewLocation(&start, &end.End),
	}
}
