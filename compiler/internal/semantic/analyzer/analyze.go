package analyzer

import (
	"compiler/ctx"
	"compiler/internal/frontend/ast"
)

type AnalyzerNode struct {
	Ctx     *ctx.CompilerContext
	Program *ast.Program
	Debug   bool
}

func NewAnalyzerNode(program *ast.Program, ctx *ctx.CompilerContext, debug bool) *AnalyzerNode {
	return &AnalyzerNode{
		Ctx:     ctx,
		Program: program,
		Debug:   debug,
	}
}
