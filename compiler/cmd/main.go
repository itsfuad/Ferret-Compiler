package main

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/frontend/parser"
	//"compiler/internal/semantic"
	"path/filepath"
	"strings"

	"compiler/internal/semantic/resolver"
	//"compiler/internal/semantic/typecheck"
	"compiler/internal/utils/fs"
	"fmt"
	"os"
)

func Compile(filePath string, debug bool) *ctx.CompilerContext {

	filePath = filepath.ToSlash(filePath)
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path: %w", err))
	}
	absPath = filepath.ToSlash(absPath)

	rootDir := filepath.Dir(absPath)
	rootDir = filepath.ToSlash(rootDir)
	relPath, err := filepath.Rel(rootDir, absPath)
	if err != nil {
		panic(fmt.Errorf("failed to get relative path: %w", err))
	}
	relPath = filepath.ToSlash(relPath)
	moduleName := filepath.Base(relPath)
	moduleName = strings.TrimSuffix(moduleName, filepath.Ext(moduleName))
	moduleName = filepath.ToSlash(moduleName)

	fmt.Printf("Compiling file: %s\n", filePath)
	
	if !fs.IsValidFile(absPath) {
		panic(fmt.Errorf("invalid file: %s", relPath))
	}
	
	context := ctx.NewCompilerContext(absPath)

	defer func() {
		context.Reports.DisplayAll()
		if r := recover(); r != nil {
			colors.ORANGE.Println(r)
		}
	}()

	p := parser.NewParser(absPath, context, true)
	program := p.Parse()

	if program == nil {
		colors.RED.Println("Failed to parse the program.")
		return context
	}

	context.AddModule(moduleName, program)

	// Run resolver
	res := resolver.NewResolver(program, context, debug)
	res.ResolveProgram()
	
	if context.Reports.HasErrors() {
		panic("")
	}
	
	colors.GREEN.Println("Resolver done!")

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
		fmt.Println("Usage: go run main.go <filename>")
		return
	}

	debug := false

	if len(os.Args) > 2 && os.Args[2] == "--debug" {
		colors.BLUE.Println("Debug mode enabled")
		debug = true
	}

	filename := os.Args[1]
	context := Compile(filename, debug)
	defer context.Destroy()
	context.PrintModules()
}
