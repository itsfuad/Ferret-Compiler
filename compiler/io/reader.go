package io

import (
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/frontend/parser"
)

func Compile(filepath string, debug bool) *ast.Program {
	p := parser.NewParser(filepath, true)

	defer func() {
		if r := recover(); r != nil {
			colors.ORANGE.Println(r)
			p.Reports.DisplayAll()
		}
	}()

	return p.Parse()
}