package parser

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/report"
	"compiler/internal/source"
)

func parseMethodDeclaration(p *Parser, startPos *source.Position, receivers []ast.Parameter) *ast.MethodDecl {
	colors.BLUE.Println("Parsing ")
	name := p.consume(lexer.IDENTIFIER_TOKEN, report.EXPECTED_METHOD_NAME)

	iden := ast.IdentifierExpr{
		Name:     name.Value,
		Location: *source.NewLocation(&name.Start, &name.End),
	}

	if len(receivers) == 0 {
		p.ctx.Reports.AddSyntaxError(p.fullPath, &iden.Location, "Expected receiver", report.PARSING_PHASE)
		return nil
	}

	if len(receivers) > 1 {
		receiver := receivers[1]
		p.ctx.Reports.AddError(p.fullPath, &receiver.Identifier.Location, "expected only one receiver", report.PARSING_PHASE)
	}

	receiver := receivers[0]

	funcLit := parseFunctionLiteral(p, &name.Start, true)

	return &ast.MethodDecl{
		Method:   &iden,
		Receiver: &receiver,
		Function: funcLit,
		Location: *source.NewLocation(startPos, funcLit.Loc().End),
	}
}
