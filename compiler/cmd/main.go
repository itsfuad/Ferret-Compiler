package main

import (
	"compiler/cmd/resolver"
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/parser"
	"fmt"
	"os"
)

func Compile(filepath string, debug bool) *ast.Program {

	if !resolver.IsValidFile(filepath) {
		panic(fmt.Errorf("invalid file: %s", filepath))
	}

	ctx := ctx.NewCompilerContext(filepath)

	p := parser.NewParser(filepath, ctx, true)

	defer func() {
		if r := recover(); r != nil {
			colors.ORANGE.Println(r)
			ctx.Reports.DisplayAll()
		}
	}()

	return p.Parse()
}

func main() {

	//filename from command line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <filename>")
		return
	}

	filename := os.Args[1]
	fmt.Printf("Compiling file: %s\n", filename)

	program := Compile(filename, true)
	fmt.Printf("Program: %v\n", program)
}
