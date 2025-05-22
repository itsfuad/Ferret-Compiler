package parser

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/frontend/lexer"
	"ferret/compiler/internal/source"
	"ferret/compiler/report"
)

// parseIfStatement parses an if statement with optional else and else-if branches
func parseIfStatement(p *Parser) ast.BlockConstruct {

	start := p.consume(lexer.IF_TOKEN, report.EXPECTED_IF) // consume 'if'

	// Parse condition (parentheses are optional)
	var condition ast.Expression
	if p.match(lexer.OPEN_PAREN) {
		p.advance() // consume '('
		condition = parseExpression(p)
		p.consume(lexer.CLOSE_PAREN, report.EXPECTED_CLOSE_PAREN)
	} else {
		condition = parseExpression(p)
	}

	if condition == nil {
		token := p.peek()
		report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Expected condition after 'if'").SetLevel(report.SYNTAX_ERROR)
		return nil
	}
	// Parse if body
	body := parseBlock(p)
	if body == nil {
		return nil
	}

	ifStmt := &ast.IfStmt{
		Condition: condition,
		Body:      body,
		Location:  *source.NewLocation(&start.Start, body.Loc().End),
	}

	if p.match(lexer.ELSE_TOKEN) {
		p.advance() // consume 'else'
		if p.match(lexer.IF_TOKEN) {
			// Parse else-if branch recursively
			stmt := parseIfStatement(p)
			ifStmt.Alternative = stmt
		} else {
			// Parse else branch
			stmt := parseBlock(p)
			ifStmt.Alternative = stmt
		}
		// Update the end position to include the else branch
		ifStmt.End = ifStmt.Alternative.Loc().End
	}

	return ifStmt
}
