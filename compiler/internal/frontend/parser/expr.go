package parser

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/frontend/lexer"
	"ferret/compiler/internal/source"
	"ferret/compiler/report"
)

// parseExpression is the entry point for expression parsing
func parseExpression(p *Parser) ast.Expression {
	return parseLogicalOr(p)
}

// parseLogicalOr handles || operator
func parseLogicalOr(p *Parser) ast.Expression {
	expr := parseLogicalAnd(p)

	for p.match(lexer.OR_TOKEN) {
		operator := p.advance()
		right := parseLogicalAnd(p)
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator,
			Right:    right,
			Location: *source.NewLocation(expr.Loc().Start, right.Loc().End),
		}
	}

	return expr
}

// parseLogicalAnd handles && operator
func parseLogicalAnd(p *Parser) ast.Expression {
	expr := parseEquality(p)

	for p.match(lexer.AND_TOKEN) {
		operator := p.advance()
		right := parseEquality(p)
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator,
			Right:    right,
			Location: *source.NewLocation(expr.Loc().Start, right.Loc().End),
		}
	}

	return expr
}

// parseEquality handles == and != operators
func parseEquality(p *Parser) ast.Expression {
	expr := parseComparison(p)

	for p.match(lexer.DOUBLE_EQUAL_TOKEN, lexer.NOT_EQUAL_TOKEN) {
		operator := p.advance()
		right := parseComparison(p)
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator,
			Right:    right,
			Location: *source.NewLocation(expr.Loc().Start, right.Loc().End),
		}
	}

	return expr
}

// parseComparison handles <, >, <=, >= operators
func parseComparison(p *Parser) ast.Expression {
	expr := parseAdditive(p)

	for p.match(lexer.LESS_TOKEN, lexer.GREATER_TOKEN, lexer.LESS_EQUAL_TOKEN, lexer.GREATER_EQUAL_TOKEN) {
		operator := p.advance()
		right := parseAdditive(p)
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator,
			Right:    right,
			Location: *source.NewLocation(expr.Loc().Start, right.Loc().End),
		}
	}

	return expr
}

// parseAdditive handles + and - operators
func parseAdditive(p *Parser) ast.Expression {
	expr := parseMultiplicative(p)

	for p.match(lexer.PLUS_TOKEN, lexer.MINUS_TOKEN) {
		operator := p.advance()
		right := parseMultiplicative(p)
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator,
			Right:    right,
			Location: *source.NewLocation(expr.Loc().Start, right.Loc().End),
		}
	}

	return expr
}

// parseMultiplicative handles *, /, and % operators
func parseMultiplicative(p *Parser) ast.Expression {
	expr := parseUnary(p)

	for p.match(lexer.MUL_TOKEN, lexer.DIV_TOKEN, lexer.MOD_TOKEN) {
		operator := p.advance()
		right := parseUnary(p)
		expr = &ast.BinaryExpr{
			Left:     expr,
			Operator: operator,
			Right:    right,
			Location: *source.NewLocation(expr.Loc().Start, right.Loc().End),
		}
	}

	return expr
}

// parseUnary handles unary operators (!, -, ++, --)
func parseUnary(p *Parser) ast.Expression {
	if p.match(lexer.NOT_TOKEN, lexer.MINUS_TOKEN) {
		operator := p.advance()
		right := parseUnary(p)
		return &ast.UnaryExpr{
			Operator: operator,
			Operand:  right,
			Location: *source.NewLocation(&operator.Start, right.Loc().End),
		}
	}

	// Handle prefix operators (++, --)
	if p.match(lexer.PLUS_PLUS_TOKEN, lexer.MINUS_MINUS_TOKEN) {
		operator := p.advance()
		// Check for consecutive operators
		if p.match(lexer.PLUS_PLUS_TOKEN, lexer.MINUS_MINUS_TOKEN) {
			errMsg := report.INVALID_CONSECUTIVE_INCREMENT
			if operator.Kind == lexer.MINUS_MINUS_TOKEN {
				errMsg = report.INVALID_CONSECUTIVE_DECREMENT
			}
			report.Add(p.filePath, source.NewLocation(&operator.Start, &operator.End), errMsg).SetLevel(report.SYNTAX_ERROR)
			return nil
		}
		operand := parseUnary(p)
		if operand == nil {
			errMsg := report.INVALID_INCREMENT_OPERAND
			if operator.Kind == lexer.MINUS_MINUS_TOKEN {
				errMsg = report.INVALID_DECREMENT_OPERAND
			}
			report.Add(p.filePath, source.NewLocation(&operator.Start, &operator.End), errMsg).SetLevel(report.SYNTAX_ERROR)
			return nil
		}

		// Check if operand already has a postfix operator
		if _, ok := operand.(*ast.PostfixExpr); ok {
			report.Add(p.filePath, source.NewLocation(&operator.Start, &operator.End), "Cannot mix prefix and postfix operators").SetLevel(report.SYNTAX_ERROR)
			return nil
		}

		return &ast.PrefixExpr{
			Operator: operator,
			Operand:  operand,
			Location: *source.NewLocation(&operator.Start, operand.Loc().End),
		}
	}

	return parsePostfix(p)
}

// parseIndexing handles array/map indexing operations
func parseIndexing(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	start := expr.Loc().Start
	p.advance() // consume '['

	index := parseExpression(p)
	if index == nil {
		token := p.peek()
		report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), report.MISSING_INDEX_EXPRESSION).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}

	end := p.consume(lexer.CLOSE_BRACKET, report.EXPECTED_CLOSE_BRACKET)
	return &ast.IndexableExpr{
		Indexable: expr,
		Index:     index,
		Location:  *source.NewLocation(start, &end.End),
	}, true
}

// parseIncDec handles postfix increment/decrement
func parseIncDec(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	operator := p.advance()
	if p.match(lexer.PLUS_PLUS_TOKEN, lexer.MINUS_MINUS_TOKEN) {
		errMsg := report.INVALID_CONSECUTIVE_INCREMENT
		if operator.Kind == lexer.MINUS_MINUS_TOKEN {
			errMsg = report.INVALID_CONSECUTIVE_DECREMENT
		}
		report.Add(p.filePath, source.NewLocation(&operator.Start, &operator.End), errMsg).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}
	return &ast.PostfixExpr{
		Operand:  expr,
		Operator: operator,
		Location: *source.NewLocation(expr.Loc().Start, &operator.End),
	}, true
}

// handlePostfixOperator handles a single postfix operator and returns the updated expression
func handlePostfixOperator(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	if p.match(lexer.PLUS_PLUS_TOKEN, lexer.MINUS_MINUS_TOKEN) {
		if _, ok := expr.(*ast.PrefixExpr); ok {
			current := p.peek()
			report.Add(p.filePath, source.NewLocation(&current.Start, &current.End), "Cannot mix prefix and postfix operators").SetLevel(report.SYNTAX_ERROR)
			return nil, false
		}
		return parseIncDec(p, expr)
	}

	if p.match(lexer.DOT_TOKEN) {
		return parseFieldAccess(p, expr)
	}

	if p.match(lexer.SCOPE_TOKEN) {
		return parseScopeResolution(p, expr)
	}

	if p.match(lexer.OPEN_PAREN) {
		return parseFunctionCall(p, expr)
	}

	if p.match(lexer.OPEN_BRACKET) {
		return parseIndexing(p, expr)
	}

	return nil, false
}

// parsePostfix handles postfix operators (++, --, [], ., (), {})
func parsePostfix(p *Parser) ast.Expression {
	expr := parsePrimary(p)
	if expr == nil {
		return nil
	}

	for {
		if newExpr, handled := handlePostfixOperator(p, expr); handled {
			if newExpr == nil {
				return nil
			}
			expr = newExpr
			continue
		}
		break
	}

	return expr
}

// parseGrouping handles parenthesized expressions
func parseGrouping(p *Parser) ast.Expression {
	p.advance() // consume '('
	expr := parseExpression(p)
	p.consume(lexer.CLOSE_PAREN, "Expected ')' after expression")
	return expr
}

// parseFunctionCall parses a function call expression
func parseFunctionCall(p *Parser, caller ast.Expression) (ast.Expression, bool) {
	start := caller.Loc().Start
	p.advance() // consume '('

	arguments := make([]ast.Expression, 0)
	// Parse arguments
	for !p.match(lexer.CLOSE_PAREN) {
		arg := parseExpression(p)
		if arg == nil {
			token := p.peek()
			report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Expected function argument").SetLevel(report.SYNTAX_ERROR)
			return nil, false
		}
		arguments = append(arguments, arg)

		if p.match(lexer.CLOSE_PAREN) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_PAREN)
			if p.match(lexer.CLOSE_PAREN) {
				report.Add(p.filePath, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED).AddHint("Remove the trailing comma").SetLevel(report.WARNING)
				break
			}
		}
	}

	end := p.consume(lexer.CLOSE_PAREN, report.EXPECTED_CLOSE_PAREN)

	return &ast.FunctionCallExpr{
		Caller:    caller,
		Arguments: arguments,
		Location:  *source.NewLocation(start, &end.End),
	}, true
}

// parsePrimary handles literals, identifiers, and parenthesized expressions
func parsePrimary(p *Parser) ast.Expression {
	switch p.peek().Kind {
	case lexer.OPEN_PAREN:
		return parseGrouping(p)
	case lexer.OPEN_BRACKET:
		return parseArrayLiteral(p)
	case lexer.NUMBER_TOKEN:
		return parseNumberLiteral(p)
	case lexer.STRING_TOKEN:
		return parseStringLiteral(p)
	case lexer.BYTE_TOKEN:
		return parseByteLiteral(p)
	case lexer.FUNCTION_TOKEN:
		start := p.advance()
		return parseFunctionLiteral(p, &start.Start, true, true)
	case lexer.AT_TOKEN:
		return parseStructLiteral(p)
	case lexer.IDENTIFIER_TOKEN:
		return parseIdentifier(p)
	}
	handleUnexpectedToken(p)
	return nil
}
