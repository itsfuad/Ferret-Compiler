package parser

import (
	"ferret/internal/frontend/ast"
	"ferret/internal/frontend/lexer"
	"ferret/internal/source"
	"ferret/report"
)

func parseMethodDeclaration(p *Parser, startPos *source.Position, receivers []ast.Parameter) *ast.MethodDecl {

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
	funcLit.ID = iden.Name

	return &ast.MethodDecl{
		Method:   &iden,
		Receiver: &receiver,
		Function: funcLit,
		Location: *source.NewLocation(startPos, funcLit.Loc().End),
	}
}
