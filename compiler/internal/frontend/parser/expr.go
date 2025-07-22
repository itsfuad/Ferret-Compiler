package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
)

// parseExpressionList parses a comma-separated list of expressions
func parseExpressionList(p *Parser, first ast.Expression) ast.ExpressionList {
	exprs := ast.ExpressionList{first}
	for p.match(lexer.COMMA_TOKEN) {
		p.advance() // consume comma
		next := parseExpression(p)
		if next == nil {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected expression after comma", report.PARSING_PHASE)
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

func parseReturnStmt(p *Parser) ast.Statement {
	start := p.consume(lexer.RETURN_TOKEN, report.EXPECTED_RETURN_KEYWORD).Start
	end := start

	// Return immediately if there's a semicolon (no return value)
	if p.match(lexer.SEMICOLON_TOKEN) {
		return &ast.ReturnStmt{
			Value:    nil,
			Location: *source.NewLocation(&start, &end),
		}
	}

	// Parse the return expression
	value := parseExpression(p)
	if value == nil {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(
			p.fullPath,
			source.NewLocation(&token.Start, &token.End),
			report.INVALID_EXPRESSION,
			report.PARSING_PHASE,
		).AddHint("Add an expression after the return keyword")

		return &ast.ReturnStmt{
			Value:    nil,
			Location: *source.NewLocation(&start, &end),
		}
	}

	end = *value.Loc().End

	// Check for unsupported multiple return values
	if p.match(lexer.COMMA_TOKEN) {
		comma := p.peek()
		p.ctx.Reports.AddSyntaxError(
			p.fullPath,
			source.NewLocation(&comma.Start, &comma.End),
			"Multiple return values are not supported",
			report.PARSING_PHASE,
		).AddHint("Functions can only return a single value")

		for p.match(lexer.COMMA_TOKEN) {
			p.advance() // consume comma
			if expr := parseExpression(p); expr != nil {
				end = *expr.Loc().End
			}
		}
	}

	return &ast.ReturnStmt{
		Value:    &value,
		Location: *source.NewLocation(&start, &end),
	}
}

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
		left := expr // Create a copy to avoid circular reference
		expr = &ast.BinaryExpr{
			Left:     &left,
			Operator: operator,
			Right:    &right,
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
		left := expr // Create a copy to avoid circular reference
		expr = &ast.BinaryExpr{
			Left:     &left,
			Operator: operator,
			Right:    &right,
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
		left := expr // Create a copy to avoid circular reference
		expr = &ast.BinaryExpr{
			Left:     &left,
			Operator: operator,
			Right:    &right,
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
		left := expr // Create a copy to avoid circular reference
		expr = &ast.BinaryExpr{
			Left:     &left,
			Operator: operator,
			Right:    &right,
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
		left := expr // Create a copy to avoid circular reference
		expr = &ast.BinaryExpr{
			Left:     &left,
			Operator: operator,
			Right:    &right,
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
		left := expr // Create a copy to avoid circular reference
		expr = &ast.BinaryExpr{
			Left:     &left,
			Operator: operator,
			Right:    &right,
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
			Operand:  &right,
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
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&operator.Start, &operator.End), errMsg, report.PARSING_PHASE)
			return nil
		}
		operand := parseUnary(p)
		if operand == nil {
			errMsg := report.INVALID_INCREMENT_OPERAND
			if operator.Kind == lexer.MINUS_MINUS_TOKEN {
				errMsg = report.INVALID_DECREMENT_OPERAND
			}
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&operator.Start, &operator.End), errMsg, report.PARSING_PHASE)
			return nil
		}

		// Check if operand already has a postfix operator
		if _, ok := operand.(*ast.PostfixExpr); ok {
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&operator.Start, &operator.End), "Cannot mix prefix and postfix operators", report.PARSING_PHASE)
			return nil
		}

		return &ast.PrefixExpr{
			Operator: operator,
			Operand:  &operand,
			Location: *source.NewLocation(&operator.Start, operand.Loc().End),
		}
	}

	return parseCast(p)
}

// parseCast handles type cast expressions (value as Type)
func parseCast(p *Parser) ast.Expression {
	expr := parsePostfix(p)

	if p.match(lexer.AS_TOKEN) {
		asToken := p.advance()
		targetType, ok := parseType(p)
		if !ok || targetType == nil {
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&asToken.Start, &asToken.End), "Expected type after 'as' keyword", report.PARSING_PHASE)
			return expr
		}

		return &ast.CastExpr{
			Value:      &expr,
			TargetType: targetType,
			Location:   *source.NewLocation(expr.Loc().Start, targetType.Loc().End),
		}
	}

	return expr
}

// parseIndexing handles array/map indexing operations
func parseIndexing(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	start := expr.Loc().Start
	p.advance() // consume '['

	index := parseExpression(p)
	if index == nil {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), report.MISSING_INDEX_EXPRESSION, report.PARSING_PHASE)
		return nil, false
	}

	end := p.consume(lexer.CLOSE_BRACKET, report.EXPECTED_CLOSE_BRACKET)
	return &ast.IndexableExpr{
		Indexable: &expr,
		Index:     &index,
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
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&operator.Start, &operator.End), errMsg, report.PARSING_PHASE)
		return nil, false
	}
	return &ast.PostfixExpr{
		Operand:  &expr,
		Operator: operator,
		Location: *source.NewLocation(expr.Loc().Start, &operator.End),
	}, true
}

// handlePostfixOperator handles a single postfix operator and returns the updated expression
func handlePostfixOperator(p *Parser, expr ast.Expression) (ast.Expression, bool) {
	if p.match(lexer.PLUS_PLUS_TOKEN, lexer.MINUS_MINUS_TOKEN) {
		if _, ok := expr.(*ast.PrefixExpr); ok {
			current := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&current.Start, &current.End), "Cannot mix prefix and postfix operators", report.PARSING_PHASE)
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
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected function argument", report.PARSING_PHASE)
			return nil, false
		}
		arguments = append(arguments, arg)

		if p.match(lexer.CLOSE_PAREN) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_PAREN)
			if p.match(lexer.CLOSE_PAREN) {
				p.ctx.Reports.AddWarning(p.fullPath, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED, report.PARSING_PHASE).AddHint("Remove the trailing comma")
				break
			}
		}
	}

	end := p.consume(lexer.CLOSE_PAREN, report.EXPECTED_CLOSE_PAREN)

	return &ast.FunctionCallExpr{
		Caller:    &caller,
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
