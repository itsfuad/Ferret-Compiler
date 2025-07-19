package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	//"strings"

	"compiler/colors"
	"compiler/internal/ctx"

	//"compiler/internal/backend"
	"compiler/internal/config"
	"compiler/internal/frontend/parser"

	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/collector"
	"compiler/internal/semantic/resolver"
	"compiler/internal/semantic/typecheck"
)

func Compile(filePath string, isDebugEnabled bool, outputPath string) *ctx.CompilerContext {
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

	if context.Reports.HasErrors() {
		panic("Compilation stopped due to resolver errors")
	}

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

	// --- Code Generation ---
	// Generate assembly code
	// if outputPath == "" {
	// 	// Use the program name for the output file
	// 	fileName := filepath.Base(fullPath)
	// 	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	// 	outputPath = filepath.Join(filepath.Dir(fullPath), baseName+".asm")
	// }

	// err = backend.CompileToAssembly(program, context, outputPath, isDebugEnabled)
	// if err != nil {
	// 	colors.RED.Printf("Code generation failed: %v\n", err)
	// 	return context
	// }

	// if isDebugEnabled {
	// 	colors.GREEN.Println("---------- [Code Generation done] ----------")
	// }

	return context
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ferret <filename> [-debug] [-o <output>] | ferret init [path/to/project]")
		os.Exit(1)
	}

	filename, debug, initProject, initPath, outputPath := parseArgs()

	// Handle init command
	if initProject {
		projectRoot := initPath
		if projectRoot == "" {
			cwd, err := os.Getwd()
			if err != nil {
				colors.RED.Println("Failed to get current working directory:", err)
				os.Exit(1)
			}
			projectRoot = cwd
		}

		// Create the configuration file
		if err := config.CreateDefaultProjectConfig(projectRoot); err != nil {
			colors.RED.Println("Failed to initialize project configuration:", err)
			os.Exit(1)
		}
		colors.GREEN.Printf("Project configuration initialized at: %s\n", projectRoot)
		return
	}

	// Check for filename argument
	if filename == "" {
		fmt.Println("Usage: ferret <filename> [-debug] [-o <output>] | ferret init [path]")
		os.Exit(1)
	}

	if debug {
		colors.BLUE.Println("Debug mode enabled")
	}

	context := Compile(filename, debug, outputPath)

	// Only destroy and print modules if context is not nil
	if context != nil {
		defer context.Destroy()
		context.PrintModules()
	}
}
