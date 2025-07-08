package main

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/parser"
	"compiler/internal/semantic"
	"compiler/internal/semantic/resolver"
	"compiler/internal/semantic/typecheck"
	"compiler/internal/utils/fs"
	"fmt"
	"os"
)

func Compile(filepath string, debug bool) *ctx.CompilerContext {

	if !fs.IsValidFile(filepath) {
		panic(fmt.Errorf("invalid file: %s", filepath))
	}

	context := ctx.NewCompilerContext(filepath)

	p := parser.NewParser(filepath, context, true)

	defer func() {
		context.Reports.DisplayAll()
		if r := recover(); r != nil {
			colors.ORANGE.Println(r)
		}
	}()

	program := p.Parse()

	if program == nil {
		colors.RED.Println("Failed to parse the program.")
		return context
	}

	context.AddModule(ctx.LocalModuleKey(filepath), program)

	// --- Semantic Analysis: Name Resolution ---
	globalTable := semantic.NewSymbolTable(nil)
	semantic.AddPreludeSymbols(globalTable)
	res := resolver.NewResolver(context, filepath, &context.Reports, debug)
	res.Symbols = globalTable
	res.ResolveProgram(program)
	if context.Reports.HasErrors() {
		context.Reports.DisplayAll()
		return context
	}

	// --- Type Checking ---
	tc := typecheck.NewTypeChecker(globalTable, &context.Reports, debug)
	tc.SetContext(context)
	tc.CheckProgram(program)
	if context.Reports.HasErrors() {
		context.Reports.DisplayAll()
		return context
	}

	return context
}

func main() {

	//filename from command line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <filename>")
		return
	}

	debug := false

	if len(os.Args) > 2 && os.Args[2] == "--debug" {
		colors.BLUE.Println("Debug mode enabled")
		debug = true
	}

	filename := os.Args[1]
	fmt.Printf("Compiling file: %s\n", filename)

	context := Compile(filename, debug)
	defer context.Destroy()
	context.PrintModules()
}
