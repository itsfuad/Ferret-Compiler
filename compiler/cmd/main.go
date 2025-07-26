package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"compiler/cmd/flags"
	"compiler/colors"
	"compiler/internal/ctx"
	"compiler/internal/registry"

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

	// Create a temporary context to access remote module functionality
	// We need to find a fer.ret file to establish project root
	dummyFile := filepath.Join(cwd, "dummy.fer")
	context := ctx.NewCompilerContext(dummyFile)
	defer context.Destroy()

	if module == "" {
		// Install all dependencies from fer.ret
		colors.BLUE.Println("Installing all dependencies from fer.ret...")
		err := registry.InstallDependencies(context)
		if err != nil {
			colors.RED.Printf("Failed to install dependencies: %s\n", err)
			os.Exit(1)
		}
	} else {
		// Install specific module
		if !context.IsRemoteImport(module) {
			colors.RED.Printf("Invalid remote module path: %s\n", module)
			colors.YELLOW.Println("Remote modules should start with github.com/, gitlab.com/, etc.")
			os.Exit(1)
		}

		repoPath, version, _ := context.ParseRemoteImport(module)
		colors.BLUE.Printf("Installing module: %s@%s\n", repoPath, version)

		err := registry.DownloadRemoteModule(context, repoPath, version)
		if err != nil {
			colors.RED.Printf("Failed to download module: %s\n", err)
			os.Exit(1)
		}
	}
}

func handleRemoveCommand(module string) {
	if module == "" {
		colors.RED.Println("No module specified. Usage: ferret remove [module]")
		os.Exit(1)
	}

	// Get current working directory as project root
	projectRoot, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}

	// Check if fer.ret exists
	ferRetPath := filepath.Join(projectRoot, "fer.ret")
	if _, err := os.Stat(ferRetPath); os.IsNotExist(err) {
		colors.YELLOW.Println("No fer.ret file found in current directory.")
		return
	}

	// Parse dependencies from fer.ret to check if module exists
	dependencies, err := registry.ParseFerRetDependencies(projectRoot)
	if err != nil {
		colors.RED.Printf("Failed to parse fer.ret dependencies: %s\n", err)
		os.Exit(1)
	}

	// Check if the module is in dependencies
	if _, exists := dependencies[module]; !exists {
		colors.YELLOW.Printf("Module '%s' is not in fer.ret dependencies. Nothing to remove.\n", module)
		return
	}

	// Remove from fer.ret file
	err = registry.RemoveDependencyFromFerRet(ferRetPath, module)
	if err != nil {
		colors.RED.Printf("Failed to remove module from fer.ret: %s\n", err)
		os.Exit(1)
	}

	// Remove from cache if it exists
	cachePath := filepath.Join(projectRoot, ".ferret", "modules")
	if err := registry.RemoveModuleFromCache(cachePath, module); err != nil {
		colors.YELLOW.Printf("Warning: Failed to remove module from cache: %s\n", err)
	}

	// Update lockfile
	lockfilePath := filepath.Join(projectRoot, "ferret.lock")
	if err := registry.RemoveModuleFromLockfile(lockfilePath, module); err != nil {
		colors.YELLOW.Printf("Warning: Failed to remove module from lockfile: %s\n", err)
	}

	colors.GREEN.Printf("Successfully removed module: %s\n", module)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ferret <filename> [-debug] [-o <o>] | ferret init [path/to/project] | ferret get [module] | ferret remove [module]")
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
		fmt.Println("Usage: ferret <filename> [-debug] [-o <o>] | ferret init [path] | ferret get [module] | ferret remove [module]")
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
