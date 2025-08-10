package cmd

import (
	"fmt"
	"path/filepath"
	"runtime/debug"

	"ferret/colors"
	"ferret/internal/ctx"

	"ferret/internal/frontend/parser"

	"ferret/internal/semantic/analyzer"
	"ferret/internal/semantic/collector"
	"ferret/internal/semantic/resolver"
	"ferret/internal/semantic/typecheck"
)

func Compile(filePath string, isDebugEnabled bool, outputPath string) (context *ctx.CompilerContext) {
	fullPath, err := filepath.Abs(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path: %w", err))
	}

	fullPath = filepath.ToSlash(fullPath) // Ensure forward slashes for consistency

	context = ctx.NewCompilerContext(fullPath)

	defer func() {
		context.Reports.DisplayAll()
		if r := recover(); r != nil {
			colors.ORANGE.Println("PANIC occurred:", r)
			fmt.Println("Stack trace:")
			debug.PrintStack()
		}
	}()

	p := parser.NewParser(fullPath, context, true)
	program := p.Parse()

	if program == nil {
		colors.RED.Println("Failed to parse the program.")
		return context
	}

	if isDebugEnabled {
		colors.BLUE.Printf("---------- [Parsing done] ----------\n")
	}

	anz := analyzer.NewAnalyzerNode(program, context, isDebugEnabled)

	// --- Semantic Analysis ---
	// Collect symbols
	collector.CollectSymbols(anz)

	if isDebugEnabled {
		colors.BLUE.Printf("---------- [Symbol Collection done] ----------\n")
	}

	resolver.ResolveProgram(anz)

	// if context.Reports.HasErrors() {
	// 	panic("Compilation stopped due to resolver errors")
	// }

	if isDebugEnabled {
		colors.GREEN.Println("---------- [Resolver done] ----------")
	}

	typecheck.CheckProgram(anz)

	if context.Reports.HasErrors() {
		panic("Compilation stopped due to type checking errors")
	}

	if isDebugEnabled {
		colors.GREEN.Println("---------- [Type Checking done] ----------")
	}

	return context
}
