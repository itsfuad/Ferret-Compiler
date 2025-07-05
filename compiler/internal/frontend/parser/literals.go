package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
	"compiler/internal/types"
	"strings"
)

func parseNumberLiteral(p *Parser) ast.Expression {
	number := p.consume(lexer.NUMBER_TOKEN, report.EXPECTED_NUMBER)
	raw := number.Value
	value := strings.ReplaceAll(raw, "_", "") // Remove underscores
	loc := *source.NewLocation(&number.Start, &number.End)

	// Try parsing as integer first
	if types.ValidateHexadecimal(value) {
		intVal, err := types.ParseInteger(value)
		if err != nil {
			p.ctx.Reports.Add(p.filePath, &loc, report.INT_OUT_OF_RANGE, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		return &ast.IntLiteral{
			Value:    intVal,
			Raw:      raw,
			Base:     16,
			Location: loc,
		}
	}

	if types.ValidateOctal(value) {
		intVal, err := types.ParseInteger(value)
		if err != nil {
			p.ctx.Reports.Add(p.filePath, &loc, report.INT_OUT_OF_RANGE, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		return &ast.IntLiteral{
			Value:    intVal,
			Raw:      raw,
			Base:     8,
			Location: loc,
		}
	}

	if types.ValidateBinary(value) {
		intVal, err := types.ParseInteger(value)
		if err != nil {
			p.ctx.Reports.Add(p.filePath, &loc, report.INT_OUT_OF_RANGE, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
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
	if types.ValidateDecimal(value) {
		intVal, err := types.ParseInteger(value)
		if err != nil {
			p.ctx.Reports.Add(p.filePath, &loc, report.INT_OUT_OF_RANGE, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
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
	if types.ValidateFloat(value) {
		floatVal, err := types.ParseFloat(value)
		if err != nil {
			p.ctx.Reports.Add(p.filePath, &loc, report.FLOAT_OUT_OF_RANGE, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil
		}

		return &ast.FloatLiteral{
			Value:    floatVal,
			Raw:      raw,
			Location: loc,
		}
	}

	// If neither, it's an invalid number format
	p.ctx.Reports.Add(p.filePath, &loc, report.INVALID_NUMBER, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
	return nil
}

func parseStringLiteral(p *Parser) ast.Expression {
	stringLiteral := p.consume(lexer.STRING_TOKEN, report.EXPECTED_STRING)
	loc := *source.NewLocation(&stringLiteral.Start, &stringLiteral.End)

	return &ast.StringLiteral{
		Value:    stringLiteral.Value,
		Location: loc,
	}
}

func parseByteLiteral(p *Parser) ast.Expression {
	byteLiteral := p.consume(lexer.BYTE_TOKEN, report.EXPECTED_BYTE)
	loc := *source.NewLocation(&byteLiteral.Start, &byteLiteral.End)

	return &ast.ByteLiteral{
		Value:    byteLiteral.Value,
		Location: loc,
	}
}

func parseArrayLiteral(p *Parser) ast.Expression {
	start := p.advance().Start // consume '['
	elements := make([]ast.Expression, 0)

	for !p.match(lexer.CLOSE_BRACKET) {
		expr := parseExpression(p)
		if expr != nil {
			elements = append(elements, expr)
		}

		if p.match(lexer.CLOSE_BRACKET) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_BRACKET)
			if p.match(lexer.CLOSE_BRACKET) {
				p.ctx.Reports.Add(p.filePath, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED, report.PARSING_PHASE).AddHint("Remove the trailing comma").SetLevel(report.WARNING)
				break
			}
		}
	}

	end := p.consume(lexer.CLOSE_BRACKET, report.EXPECTED_CLOSE_BRACKET)

	// at least one element required
	if len(elements) == 0 {
		peek := p.peek()
		p.ctx.Reports.Add(p.filePath, source.NewLocation(&peek.Start, &peek.End), report.ARRAY_EMPTY, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil
	}

	return &ast.ArrayLiteralExpr{
		Elements: elements,
		Location: *source.NewLocation(&start, &end.End),
	}
}
