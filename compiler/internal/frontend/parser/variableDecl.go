package parser

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/frontend/lexer"
	"ferret/compiler/internal/source"
	"ferret/compiler/report"
	"ferret/compiler/types"
	"fmt"
)

func parseIdentifiers(p *Parser) ([]*ast.VariableToDeclare, int) {

	variables := make([]*ast.VariableToDeclare, 0)
	varCount := 0

	for {
		if !p.check(lexer.IDENTIFIER_TOKEN) {
			token := p.peek()
			report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), report.MISSING_NAME).SetLevel(report.SYNTAX_ERROR)
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
			report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), report.MISSING_TYPE_NAME).SetLevel(report.SYNTAX_ERROR)
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

func parseInitializers(p *Parser) ([]ast.Expression, bool) {

	values := make([]ast.Expression, 0)

	if p.match(lexer.EQUALS_TOKEN) {
		p.advance()
		for {
			value := parseExpression(p)
			if value == nil {
				token := p.peek()
				report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "Expected value after '=', got invalid expression").SetLevel(report.SYNTAX_ERROR)
				return nil, false
			}
			values = append(values, value)

			if p.peek().Kind != lexer.COMMA_TOKEN {
				break
			}
			p.advance()
		}
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
	report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), report.MISMATCHED_VARIABLE_AND_TYPE_COUNT+fmt.Sprintf(": Expected %d types, got %d", varCount, len(types))).SetLevel(report.SYNTAX_ERROR)
	return false
}

func parseVarDecl(p *Parser) ast.Statement {
	token := p.advance() // consume let/const

	isConst := token.Kind == lexer.CONST_TOKEN

	variables, varCount := parseIdentifiers(p)
	if variables == nil {
		pos := p.peek()
		report.Add(p.filePath, source.NewLocation(&pos.Start, &pos.End), "no variables found").SetLevel(report.SYNTAX_ERROR)
		return nil
	}

	parsedTypes, ok := parseTypeAnnotations(p)
	if !ok || !assignTypes(p, variables, parsedTypes, varCount) {
		return nil
	}

	values, ok := parseInitializers(p)
	if !ok {
		return nil
	}

	// when no types provides, we must initialize
	if len(parsedTypes) == 0 {
		if len(values) == 0 {
			token := p.peek()
			report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "cannot infer types without initializers").AddHint("👈😃 Add initializers to the variables").SetLevel(report.NORMAL_ERROR)
			return nil
		}
	}

	if len(values) > varCount {
		token := p.peek()
		report.Add(p.filePath, source.NewLocation(&token.Start, &token.End), "values cannot be more than the number of variables").SetLevel(report.SYNTAX_ERROR)
		return nil
	}

	// Add variables to symbol table
	for _, v := range variables {

		var typename types.TYPE_NAME
		if v.ExplicitType != nil {
			typename = v.ExplicitType.Type()
		}

		fmt.Printf("var %s's typename: %+v\n", v.Identifier.Name, typename)
	}

	return &ast.VarDeclStmt{
		Variables:    variables,
		Initializers: values,
		IsConst:      isConst,
		Location:     *source.NewLocation(&token.Start, variables[len(variables)-1].Identifier.Loc().End),
	}
}
