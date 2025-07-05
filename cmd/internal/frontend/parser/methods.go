package parser

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"compiler/report"
)

func parseMethodDeclaration(p *Parser, startPos *source.Position, receivers []ast.Parameter) *ast.MethodDecl {
	colors.BLUE.Println("Parsing ")
	name := p.consume(lexer.IDENTIFIER_TOKEN, report.EXPECTED_METHOD_NAME)

	iden := ast.IdentifierExpr{
		Name:     name.Value,
		Location: *source.NewLocation(&name.Start, &name.End),
	}

	if len(receivers) == 0 {
		p.Reports.Add(p.filePath, &iden.Location, "Expected receiver").SetLevel(report.SYNTAX_ERROR)
		return nil
	}

	if len(receivers) > 1 {
		receiver := receivers[1]
		p.Reports.Add(p.filePath, &receiver.Identifier.Location, "expected only one receiver").SetLevel(report.NORMAL_ERROR)
	}

	receiver := receivers[0]

	funcLit := parseFunctionLiteral(p, &name.Start, false, true)

	return &ast.MethodDecl{
		Method:   iden,
		Receiver: &receiver,
		Function: funcLit,
		Location: *source.NewLocation(startPos, funcLit.Loc().End),
	}
}
