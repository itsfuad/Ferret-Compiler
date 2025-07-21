package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
)

func parseIdentifier(p *Parser) *ast.IdentifierExpr {
	token := p.consume(lexer.IDENTIFIER_TOKEN, "Expected identifier")

	// Create the identifier expression
	identifier := &ast.IdentifierExpr{
		Name:     token.Value,
		Location: *source.NewLocation(&token.Start, &token.End),
	}

	return identifier
}
