package parser

import (
	"fmt"

	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"compiler/report"
)

func parseIdentifiers(p *Parser) ([]*ast.VariableToDeclare, int) {

	variables := make([]*ast.VariableToDeclare, 0)
	varCount := 0

	for {
		if !p.check(lexer.IDENTIFIER_TOKEN) {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), report.MISSING_NAME, report.PARSING_PHASE)
			return nil, 0
		}
		identifierName := p.advance()
		identifier := &ast.VariableToDeclare{
			Identifier: &ast.IdentifierExpr{
				Name:     identifierName.Value,
				Location: *source.NewLocation(&identifierName.Start, &identifierName.End),
			},
		}
		variables = append(variables, identifier)
		varCount++

		if p.peek().Kind != lexer.COMMA_TOKEN {
			break
		}
		p.advance()
	}
	return variables, varCount
}

// parseTypeAnnotations parses the type annotations for the variables
// it returns a list of types and a boolean indicating if the parsing was successful
func parseTypeAnnotations(p *Parser) ([]ast.DataType, bool) {
	if p.peek().Kind != lexer.COLON_TOKEN {
		return nil, true
	}

	p.advance()
	types := make([]ast.DataType, 0)
	for {
		typeNode, ok := parseType(p)
		if !ok {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), report.EXPECTED_TYPE, report.PARSING_PHASE)
			return nil, false
		}
		types = append(types, typeNode)

		if p.peek().Kind != lexer.COMMA_TOKEN {
			break
		}
		p.advance()
	}
	return types, true
}

func parseValueList(p *Parser) ([]ast.Expression, bool) {
	values := make([]ast.Expression, 0)

	// Parse comma-separated values
	for {
		value := parseExpression(p)
		if value == nil {
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "Expected valid expression", report.PARSING_PHASE)
			return nil, false
		}
		values = append(values, value)

		if p.peek().Kind != lexer.COMMA_TOKEN {
			break
		}
		p.advance()
	}

	return values, true
}

func assignTypes(p *Parser, variables []*ast.VariableToDeclare, types []ast.DataType, varCount int) bool {
	if len(types) == 0 {
		return true
	}
	if len(types) == 1 {
		for i := range variables {
			variables[i].ExplicitType = types[0]
		}
		return true
	}
	if len(types) == varCount {
		for i := range variables {
			variables[i].ExplicitType = types[i]
		}
		return true
	}
	token := p.peek()
	p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), report.MISMATCHED_VARIABLE_AND_TYPE_COUNT+fmt.Sprintf(": Expected %d types, got %d", varCount, len(types)), report.PARSING_PHASE)
	return false
}

func parseVarDecl(p *Parser) ast.Statement {
	token := p.advance() // consume let/const

	isConst := token.Kind == lexer.CONST_TOKEN

	variables, varCount := parseIdentifiers(p)
	if variables == nil {
		pos := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&pos.Start, &pos.End), "no variables found", report.PARSING_PHASE)
		return nil
	}

	var parsedTypes []ast.DataType
	var values []ast.Expression
	var ok bool

	// Check the next token to determine parsing path
	nextToken := p.peek()
	switch nextToken.Kind {
	case lexer.COLON_TOKEN:
		// Path 1: : types... [= values, ...]
		parsedTypes, ok = parseTypeAnnotations(p)
		if !ok || !assignTypes(p, variables, parsedTypes, varCount) {
			return nil
		}

		// Check what comes after types
		nextToken := p.peek()
		switch nextToken.Kind {
		case lexer.EQUALS_TOKEN:
			// Optionally parse initializers with =
			p.advance() // consume =
			values, ok = parseValueList(p)
			if !ok {
				return nil
			}
		case lexer.WALRUS_TOKEN:
			// Error: cannot use := after explicit types
			token := p.peek()
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "cannot use ':=' after explicit types, use '=' instead", report.PARSING_PHASE)
			return nil
		}
		// If neither = nor :=, that's fine (declaration only)

	case lexer.WALRUS_TOKEN:
		// Path 2: := values... (type inference)
		p.advance() // consume :=
		values, ok = parseValueList(p)
		if !ok {
			return nil
		}

		// No explicit types for type inference
		parsedTypes = []ast.DataType{}

	default:
		// Neither : nor := found
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "expected ':' for type annotation or ':=' for type inference", report.PARSING_PHASE)
		return nil
	}

	// Validation: if no types and no values, error
	if len(parsedTypes) == 0 && len(values) == 0 {
		token := p.peek()
		p.ctx.Reports.AddError(p.fullPath, source.NewLocation(&token.Start, &token.End), "cannot infer types without initializers", report.PARSING_PHASE).AddHint("👈😃 Use ':=' to initialize variables for type inference")
		return nil
	}

	// Validation: values cannot exceed variable count
	if len(values) > varCount {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), "values cannot be more than the number of variables", report.PARSING_PHASE)
		return nil
	}

	return &ast.VarDeclStmt{
		Variables:    variables,
		Initializers: values,
		IsConst:      isConst,
		Location:     *source.NewLocation(&token.Start, variables[len(variables)-1].Identifier.Loc().End),
	}
}
