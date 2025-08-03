package analyzer

import (
	"ferret/compiler/internal/ctx"
	"ferret/compiler/internal/frontend/ast"
)

type AnalyzerNode struct {
	Ctx         *ctx.CompilerContext
	Program     *ast.Program
	Debug       bool
	UsedImports map[string]bool // Track which imports are used in this file
}

func NewAnalyzerNode(program *ast.Program, ctx *ctx.CompilerContext, debug bool) *AnalyzerNode {
	return &AnalyzerNode{
		Ctx:     ctx,
		Program: program,
		Debug:   debug,
	}
}
