package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/config"
	"compiler/internal/frontend/parser"

	//"compiler/internal/semantic"
	// "strings"

	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/resolver"
	//"compiler/internal/semantic/typecheck"
)

func Compile(filePath string, isDebugEnabled bool) *ctx.CompilerContext {
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

	// Run resolver
	anz := analyzer.NewAnalyzerNode(program, context, isDebugEnabled)
	resolver.ResolveProgram(anz)

	if context.Reports.HasErrors() {
		panic("Compilation stopped due to errors")
	}

	if isDebugEnabled {
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

func parseArgs() (string, bool, bool, string) {
	var filename string
	var debug bool
	var initProject bool
	var initPath string

	args := os.Args[1:]

	for i, arg := range args {
		switch arg {
		case "--debug":
			debug = true
		case "init":
			initProject = true
			// Check if next argument is a path
			if i+1 < len(args) && args[i+1][:1] != "-" {
				initPath = args[i+1]
			}
		default:
			// If it's not a flag and we haven't set filename yet, this is the filename
			if !initProject && filename == "" && arg[:1] != "-" {
				filename = arg
			}
		}
	}

	return filename, debug, initProject, initPath
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ferret <filename> [--debug] | ferret init [path]")
		os.Exit(1)
	}

	filename, debug, initProject, initPath := parseArgs()

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
		fmt.Println("Usage: ferret <filename> [--debug] | ferret init [path]")
		os.Exit(1)
	}

	if debug {
		colors.BLUE.Println("Debug mode enabled")
	}

	context := Compile(filename, debug)

	// Only destroy and print modules if context is not nil
	if context != nil {
		defer context.Destroy()
		context.PrintModules()
	}
}
