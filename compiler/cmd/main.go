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
	"strings"
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

	if module == "" {
		colors.RED.Println("No module specified. Usage: ferret get <module>")
		colors.YELLOW.Println("Example: ferret get github.com/user/repo@v1.0.0")
		os.Exit(1)
	}

	// Install specific module
	err = installRemoteModule(projectRoot, module)
	if err != nil {
		colors.RED.Printf("Failed to install module: %s\n", err)
		os.Exit(1)
	}
}

// installRemoteModule installs a remote module and its dependencies
func installRemoteModule(projectRoot, moduleSpec string) error {
	// Parse the module specification (might include version like @v1.0.0)
	_, requestedVersion, repoName, err := registry.ParseRemoteImport(moduleSpec)
	if err != nil {
		return fmt.Errorf("invalid module specification: %w", err)
	}

	colors.BLUE.Printf("Installing module: %s", moduleSpec)
	if requestedVersion != "latest" {
		colors.BLUE.Printf(" (version: %s)", requestedVersion)
	}
	colors.BLUE.Println()

	// Check if the module exists on GitHub and get the actual version
	actualVersion, err := registry.CheckRemoteModuleExists(repoName, requestedVersion)
	if err != nil {
		return fmt.Errorf("module not found: %w", err)
	}

	colors.GREEN.Printf("Found version: %s\n", actualVersion)

	// Set up cache path
	cachePath := filepath.Join(projectRoot, ".ferret", "modules")
	err = os.MkdirAll(cachePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Check if already cached
	if registry.IsModuleCached(cachePath, repoName, actualVersion) {
		colors.YELLOW.Printf("Module %s@%s is already cached\n", repoName, actualVersion)
	} else {
		// Download and cache the module
		err = registry.DownloadRemoteModule(projectRoot, repoName, actualVersion, cachePath)
		if err != nil {
			return fmt.Errorf("failed to download module: %w", err)
		}
	}

	// Update fer.ret with the dependency using full repo path
	fullRepoPath := "github.com/" + repoName
	err = registry.WriteFerRetDependency(projectRoot, fullRepoPath, actualVersion, "")
	if err != nil {
		return fmt.Errorf("failed to update fer.ret: %w", err)
	}

	colors.GREEN.Printf("Successfully installed %s@%s\n", repoName, actualVersion)

	// Parse the downloaded module for its dependencies and install them recursively
	err = installTransitiveDependencies(projectRoot, repoName, cachePath)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to install transitive dependencies: %s\n", err)
		// Don't fail the entire installation for transitive dependency issues
	}

	return nil
}

// installTransitiveDependencies finds and installs dependencies of a downloaded module
func installTransitiveDependencies(projectRoot, parentRepoName, cachePath string) error {
	// We need to find the version of the parent repo from fer.ret to access its cache
	dependencies, err := registry.ReadFerRetDependencies(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to read dependencies: %w", err)
	}

	parentFullPath := "github.com/" + parentRepoName
	parentDep, exists := dependencies[parentFullPath]
	if !exists {
		return fmt.Errorf("parent module %s not found in dependencies", parentRepoName)
	}

	moduleDir := filepath.Join(cachePath, "github.com", parentRepoName+"@"+parentDep.Version)

	// Find all .fer files in the module
	ferFiles, err := findFerretFiles(moduleDir)
	if err != nil {
		return fmt.Errorf("failed to scan module files: %w", err)
	}

	// Extract remote imports from all .fer files
	remoteImports := make(map[string]bool) // Use map to avoid duplicates

	for _, ferFile := range ferFiles {
		imports, err := extractRemoteImports(ferFile)
		if err != nil {
			colors.YELLOW.Printf("Warning: Failed to parse %s: %s\n", ferFile, err)
			continue
		}

		for _, imp := range imports {
			remoteImports[imp] = true
		}
	}

	// Install each unique remote dependency
	for importPath := range remoteImports {
		err := installTransitiveDependency(projectRoot, importPath, parentRepoName)
		if err != nil {
			colors.YELLOW.Printf("Warning: Failed to install transitive dependency %s: %s\n", importPath, err)
		}
	}

	return nil
}

// installTransitiveDependency installs a single transitive dependency
func installTransitiveDependency(projectRoot, importPath, parentRepoName string) error {
	// Parse the import path
	_, requestedVersion, repoName, err := registry.ParseRemoteImport(importPath)
	if err != nil {
		return fmt.Errorf("invalid import path: %w", err)
	}

	// Check if already installed using full repo path
	dependencies, err := registry.ReadFerRetDependencies(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to read dependencies: %w", err)
	}

	fullRepoPath := "github.com/" + repoName
	if _, exists := dependencies[fullRepoPath]; exists {
		colors.BLUE.Printf("Dependency %s already installed (required by %s)\n", repoName, parentRepoName)
		return nil
	}

	colors.BLUE.Printf("Installing transitive dependency: %s (used by %s)\n", repoName, parentRepoName)

	// Check if the module exists and get version
	actualVersion, err := registry.CheckRemoteModuleExists(repoName, requestedVersion)
	if err != nil {
		return fmt.Errorf("transitive dependency not found: %w", err)
	}

	// Set up cache path
	cachePath := filepath.Join(projectRoot, ".ferret", "modules")

	// Download if not cached
	if !registry.IsModuleCached(cachePath, repoName, actualVersion) {
		err = registry.DownloadRemoteModule(projectRoot, repoName, actualVersion, cachePath)
		if err != nil {
			return fmt.Errorf("failed to download transitive dependency: %w", err)
		}
	}

	// Update fer.ret with "used by" comment using full repo path
	comment := fmt.Sprintf("used by %s", parentRepoName)
	err = registry.WriteFerRetDependency(projectRoot, fullRepoPath, actualVersion, comment)
	if err != nil {
		return fmt.Errorf("failed to update fer.ret for transitive dependency: %w", err)
	}

	colors.GREEN.Printf("Successfully installed transitive dependency %s@%s\n", repoName, actualVersion)

	// Recursively install dependencies of this dependency (prevent infinite recursion with depth limit)
	// For now, we'll limit to one level of transitive dependencies to avoid complexity
	// TODO: Implement proper cycle detection if deeper recursion is needed

	return nil
}

// findFerretFiles recursively finds all .fer files in a directory
func findFerretFiles(dir string) ([]string, error) {
	var ferFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".fer") {
			ferFiles = append(ferFiles, path)
		}

		return nil
	})

	return ferFiles, err
}

// extractRemoteImports parses a .fer file and extracts remote import statements
func extractRemoteImports(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var remoteImports []string
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for import statements: import "github.com/..."
		if strings.HasPrefix(line, "import") && strings.Contains(line, "github.com/") {
			// Extract the import path from quotes
			start := strings.Index(line, `"`)
			if start == -1 {
				continue
			}
			end := strings.Index(line[start+1:], `"`)
			if end == -1 {
				continue
			}

			importPath := line[start+1 : start+1+end]

			// Check if it's a remote import
			if strings.HasPrefix(importPath, "github.com/") {
				remoteImports = append(remoteImports, importPath)
			}
		}
	}

	return remoteImports, nil
}

// func handleRemoveCommand(module string) {
// 	if module == "" {
// 		colors.RED.Println("No module specified. Usage: ferret remove [module]")
// 		os.Exit(1)
// 	}

// 	// Get current working directory as project root
// 	projectRoot, err := os.Getwd()
// 	if err != nil {
// 		colors.RED.Println(err)
// 		os.Exit(1)
// 	}

// 	// Check if fer.ret exists
// 	ferRetPath := filepath.Join(projectRoot, "fer.ret")
// 	if _, err := os.Stat(ferRetPath); os.IsNotExist(err) {
// 		colors.YELLOW.Println("No fer.ret file found in current directory.")
// 		return
// 	}

// 	// Parse dependencies from fer.ret to check if module exists
// 	dependencies, err := registry.ParseFerRetDependencies(projectRoot)
// 	if err != nil {
// 		colors.RED.Printf("Failed to parse fer.ret dependencies: %s\n", err)
// 		os.Exit(1)
// 	}

// 	// Check if the module is in dependencies
// 	if _, exists := dependencies[module]; !exists {
// 		colors.YELLOW.Printf("Module '%s' is not in fer.ret dependencies. Nothing to remove.\n", module)
// 		return
// 	}

// 	// Remove from fer.ret file
// 	err = registry.RemoveDependency(ferRetPath, module)
// 	if err != nil {
// 		colors.RED.Printf("Failed to remove module from fer.ret: %s\n", err)
// 		os.Exit(1)
// 	}

// 	// Remove from cache if it exists
// 	cachePath := filepath.Join(projectRoot, ".ferret", "modules")
// 	if err := registry.RemoveModuleFromCache(cachePath, module); err != nil {
// 		colors.YELLOW.Printf("Warning: Failed to remove module from cache: %s\n", err)
// 	}

// 	// Update lockfile
// 	lockfilePath := filepath.Join(projectRoot, "ferret.lock")
// 	if err := registry.RemoveModuleFromLockfile(lockfilePath, module); err != nil {
// 		colors.YELLOW.Printf("Warning: Failed to remove module from lockfile: %s\n", err)
// 	}

// 	colors.GREEN.Printf("Successfully removed module: %s\n", module)
// }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ferret <filename> [-debug] [-o <o>] | ferret init [path/to/project] | ferret get [module] | ferret remove [module] | version 0.0.1")
		os.Exit(1)
	}

	args := flags.ParseArgs()

	// // Handle remove command
	// if args.RemoveCommand {
	// 	handleRemoveCommand(args.RemoveModule)
	// 	return
	// }

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
		fmt.Println("Usage: ferret <filename> [-debug] [-o <o>] | ferret init [path] | ferret get [module] | ferret remove [module] | ferret version 0.0.1")
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
