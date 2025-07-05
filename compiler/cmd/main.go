package main

import (
	"compiler/cmd/resolver"
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/parser"
	"fmt"
	"os"
)

func Compile(filepath string, debug bool) *ctx.CompilerContext {

	if !resolver.IsValidFile(filepath) {
		panic(fmt.Errorf("invalid file: %s", filepath))
	}

	context := ctx.NewCompilerContext(filepath)

	p := parser.NewParser(filepath, context, true)

	defer func() {
		if r := recover(); r != nil {
			colors.ORANGE.Println(r)
			context.Reports.DisplayAll()
			os.Exit(-1)
		}
	}()

	program := p.Parse()

	if program == nil {
		colors.RED.Println("Failed to parse the program.")
		return context
	}

	context.AddModule(ctx.LocalModuleKey(filepath), program)

	return context
}

func main() {

	//filename from command line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <filename>")
		return
	}

	filename := os.Args[1]
	fmt.Printf("Compiling file: %s\n", filename)

	context := Compile(filename, true)
	fmt.Printf("Compiled: %v\n", context.ModuleNames())
}
