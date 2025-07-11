package main

import (
	"fmt"
	"os"
	"path/filepath"
	rtdebug "runtime/debug"

	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/parser"

	//"compiler/internal/semantic"
	// "strings"

	"compiler/internal/semantic/resolver"
	//"compiler/internal/semantic/typecheck"
)

func Compile(filePath string, debug bool) *ctx.CompilerContext {
	fullPath, err := filepath.Abs(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path: %w", err))
	}

	fullPath = filepath.ToSlash(fullPath) // Ensure forward slashes for consistency

	context := ctx.NewCompilerContext(fullPath)

	defer func() {
		context.Reports.DisplayAll()
		if r := recover(); r != nil {
			colors.ORANGE.Println("PANIC occurred:", r)
			fmt.Println("Stack trace:")
			rtdebug.PrintStack()
		}
	}()

	p := parser.NewParser(fullPath, context, true)
	program := p.Parse()

	if program == nil {
		colors.RED.Println("Failed to parse the program.")
		return context
	}

	if debug {
		colors.BLUE.Printf("---------- [Parsing done] ----------\n")
	}

	// Run resolver
	res := resolver.NewResolver(program, context, debug)
	res.ResolveProgram()

	if context.Reports.HasErrors() {
		panic("Compilation stopped due to errors")
	}

	if debug {
		colors.GREEN.Println("---------- [Resolver done] ----------")
	}

	// // --- Type Checking ---
	// // Pass resolver's symbol tables and alias map to typechecker
	// typeChecker := typecheck.NewTypeChecker(program, context, debug)
	// typeChecker.CheckProgram(program)
	// if context.Reports.HasErrors() {
	// 	context.Reports.DisplayAll()
	// 	return context
	// }

	return context
}

func main() {

	//filename from command line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: ferret <filename> [OPTION]")
		return
	}

	debug := false

	if len(os.Args) > 2 && os.Args[2] == "--debug" {
		colors.BLUE.Println("Debug mode enabled")
		debug = true
	}

	filename := os.Args[1]

	context := Compile(filename, debug)

	// Only destroy and print modules if context is not nil
	if context != nil {
		defer context.Destroy()
		context.PrintModules()
	}
}
