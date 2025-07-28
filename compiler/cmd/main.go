package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"compiler/cmd/flags"
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/modules"

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

// handleGetCommand handles the "ferret get" command
func handleGetCommand(module string) {
	// Get current working directory to find project root
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}

	// Find project root by looking for fer.ret
	// FindProjectRoot expects a file path, so we create a dummy file path in the current directory
	dummyFilePath := filepath.Join(cwd, "dummy.fer")
	projectRoot, err := config.FindProjectRoot(dummyFilePath)
	if err != nil {
		colors.RED.Printf("Could not find project root (fer.ret file): %s\n", err)
		colors.YELLOW.Println("Make sure you're in a Ferret project directory or run 'ferret init' first")
		os.Exit(1)
	}

	// Load and validate project configuration
	projectConfig, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		colors.RED.Printf("Error loading project configuration: %v\n", err)
		os.Exit(1)
	}

	// ✅ SECURITY CHECK: Check if remote imports are enabled
	if !projectConfig.Remote.Enabled {
		colors.RED.Println("❌ Remote module imports are disabled in this project.")
		colors.YELLOW.Println("To enable remote imports, set 'enabled = true' in the [remote] section of fer.ret")
		os.Exit(1)
	}

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf("Failed to create dependency manager: %s\n", err)
		os.Exit(1)
	}

	if module == "" {
		// No module specified, install all dependencies from fer.ret
		colors.BLUE.Println("No module specified. Installing all dependencies from fer.ret...")
		err = dm.InstallAllDependencies()
		if err != nil {
			colors.RED.Printf("Failed to install dependencies: %s\n", err)
			os.Exit(1)
		}
		colors.GREEN.Println("All dependencies installed successfully!")
		return
	}

	// Install specific module
	err = dm.InstallDirectDependency(module, "")
	if err != nil {
		colors.RED.Printf("Failed to install module: %s\n", err)
		os.Exit(1)
	}

	colors.GREEN.Printf("Successfully installed %s\n", module)
}

// handleRemoveCommand handles the "ferret remove" command
func handleRemoveCommand(module string) {
	// Get current working directory to find project root
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}

	// Find project root by looking for fer.ret
	dummyFilePath := filepath.Join(cwd, "dummy.fer")
	projectRoot, err := config.FindProjectRoot(dummyFilePath)
	if err != nil {
		colors.RED.Printf("Could not find project root (fer.ret file): %s\n", err)
		colors.YELLOW.Println("Make sure you're in a Ferret project directory or run 'ferret init' first")
		os.Exit(1)
	}

	// Load and validate project configuration
	projectConfig, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		colors.RED.Printf("Error loading project configuration: %v\n", err)
		os.Exit(1)
	}

	// ✅ SECURITY CHECK: Check if remote imports are enabled
	if !projectConfig.Remote.Enabled {
		colors.RED.Println("❌ Remote module imports are disabled in this project.")
		colors.YELLOW.Println("To enable remote imports, set 'enabled = true' in the [remote] section of fer.ret")
		os.Exit(1)
	}

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf("Failed to create dependency manager: %s\n", err)
		os.Exit(1)
	}

	if module == "" {
		colors.RED.Println("No module specified. Usage: ferret remove <module>")
		colors.YELLOW.Println("Example: ferret remove github.com/user/repo")
		os.Exit(1)
	}

	// Remove the dependency
	err = dm.RemoveDependency(module)
	if err != nil {
		colors.RED.Printf("Failed to remove module: %s\n", err)
		os.Exit(1)
	}

	colors.GREEN.Printf("Successfully removed %s\n", module)
}

// handleListCommand handles the "ferret list" command
func handleListCommand() {
	// Get current working directory to find project root
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}

	// Find project root by looking for fer.ret
	dummyFilePath := filepath.Join(cwd, "dummy.fer")
	projectRoot, err := config.FindProjectRoot(dummyFilePath)
	if err != nil {
		colors.RED.Printf("Could not find project root (fer.ret file): %s\n", err)
		colors.YELLOW.Println("Make sure you're in a Ferret project directory or run 'ferret init' first")
		os.Exit(1)
	}

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf("Failed to create dependency manager: %s\n", err)
		os.Exit(1)
	}

	// List dependencies
	err = dm.ListDependencies()
	if err != nil {
		colors.RED.Printf("Failed to list dependencies: %s\n", err)
		os.Exit(1)
	}
}

// handleCleanupCommand handles the "ferret cleanup" command
func handleCleanupCommand() {
	// Get current working directory to find project root
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}

	// Find project root by looking for fer.ret
	dummyFilePath := filepath.Join(cwd, "dummy.fer")
	projectRoot, err := config.FindProjectRoot(dummyFilePath)
	if err != nil {
		colors.RED.Printf("Could not find project root (fer.ret file): %s\n", err)
		colors.YELLOW.Println("Make sure you're in a Ferret project directory or run 'ferret init' first")
		os.Exit(1)
	}

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf("Failed to create dependency manager: %s\n", err)
		os.Exit(1)
	}

	// Cleanup unused dependencies
	err = dm.CleanupUnusedDependencies()
	if err != nil {
		colors.RED.Printf("Failed to cleanup dependencies: %s\n", err)
		os.Exit(1)
	}

	colors.GREEN.Println("Cleanup completed successfully!")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ferret <filename> [-debug] [-o <output>] | ferret init [path/to/project] | ferret get [module] | ferret remove [module] | ferret list | ferret cleanup | version 0.0.1")
		os.Exit(1)
	}

	args := flags.ParseArgs()

	// Handle remove command
	if args.RemoveCommand {
		handleRemoveCommand(args.RemoveModule)
		return
	}

	// Handle get command
	if args.GetCommand {
		handleGetCommand(args.GetModule)
		return
	}

	// Handle list command
	if args.ListCommand {
		handleListCommand()
		return
	}

	// Handle cleanup command
	if args.CleanupCommand {
		handleCleanupCommand()
		return
	}

	// Handle init command
	if args.InitProject {
		projectRoot := args.InitPath
		if projectRoot == "" {
			cwd, err := os.Getwd()
			if err != nil {
				colors.RED.Println(err)
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
	if args.Filename == "" {
		fmt.Println("Usage: ferret <filename> [-debug] [-o <output>] | ferret init [path] | ferret get [module] | ferret remove [module] | ferret list | ferret cleanup | version 0.0.1")
		os.Exit(1)
	}

	if args.Debug {
		colors.BLUE.Println("Debug mode enabled")
	}

	context := Compile(args.Filename, args.Debug, args.OutputPath)

	// Only destroy and print modules if context is not nil
	if context != nil {
		defer context.Destroy()
		context.PrintModules()
	}
}
