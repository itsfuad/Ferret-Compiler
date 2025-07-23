package ctx

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"compiler/colors"
	"compiler/internal/config"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
)

var contextCreated = false

// ModulePhase represents the current processing phase of a module
type ModulePhase int

const (
	PhaseNotStarted  ModulePhase = iota
	PhaseParsed                  // Module has been parsed into AST
	PhaseCollected               // Symbols have been collected
	PhaseResolved                // Symbols have been resolved
	PhaseTypeChecked             // Type checking completed
)

func (p ModulePhase) String() string {
	switch p {
	case PhaseNotStarted:
		return "Not Started"
	case PhaseParsed:
		return "Parsed"
	case PhaseCollected:
		return "Collected"
	case PhaseResolved:
		return "Resolved"
	case PhaseTypeChecked:
		return "Type Checked"
	default:
		return "Unknown"
	}
}

type Module struct {
	AST         *ast.Program
	SymbolTable *SymbolTable
	Phase       ModulePhase // Current processing phase
	IsBuiltin   bool        // Whether this is a builtin module
}

type CompilerContext struct {
	EntryPoint string             // Entry point file
	Builtins   *SymbolTable       // Built-in symbols, e.g., "i32", "f64", "str", etc.
	Modules    map[string]*Module // key: import path
	Reports    report.Reports
	// Project configuration
	ProjectConfig *config.ProjectConfig
	ProjectRoot   string
	ModulesPath   string // Path to system built-in modules

	// Remote module cache path (.ferret/modules)
	RemoteCachePath string

	// Dependency graph: key is importer, value is list of imported module keys (as strings)
	DepGraph map[string][]string

	// Track modules that are currently being parsed to prevent infinite recursion
	_parsingModules map[string]bool
	// Keep track of the parsing stack to show cycle paths
	_parsingStack []string
}

func (c *CompilerContext) FullPathToImportPath(fullPath string) string {
	// Check if this is a built-in module file
	if c.isBuiltinModuleFile(fullPath) {
		return c.getBuiltinModuleImportPath(fullPath)
	}

	relPath, err := filepath.Rel(c.ProjectRoot, fullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return ""
	}
	relPath = filepath.ToSlash(relPath)
	moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))

	// Use project name from configuration instead of folder name
	projectName := ""
	if c.ProjectConfig != nil {
		projectName = c.ProjectConfig.Name
	}
	if projectName == "" {
		// Fallback to folder name if project name is not available
		projectName = filepath.Base(c.ProjectRoot)
	}

	return projectName + "/" + moduleName
}

// IsBuiltinModuleFile checks if the given file path is within the built-in modules directory
func (c *CompilerContext) IsBuiltinModuleFile(fullPath string) bool {
	return c.isBuiltinModuleFile(fullPath)
}

// isBuiltinModuleFile checks if the given file path is within the built-in modules directory
func (c *CompilerContext) isBuiltinModuleFile(fullPath string) bool {
	if c.ModulesPath == "" {
		return false
	}

	absModulesPath, _ := filepath.Abs(c.ModulesPath)
	absFilePath, _ := filepath.Abs(fullPath)

	relPath, err := filepath.Rel(absModulesPath, absFilePath)
	return err == nil && !strings.HasPrefix(relPath, "..")
}

// getBuiltinModuleImportPath generates the import path for built-in module files
func (c *CompilerContext) getBuiltinModuleImportPath(fullPath string) string {
	if c.ModulesPath == "" {
		return ""
	}

	absModulesPath, _ := filepath.Abs(c.ModulesPath)
	absFilePath, _ := filepath.Abs(fullPath)

	relPath, err := filepath.Rel(absModulesPath, absFilePath)
	if err != nil {
		return ""
	}

	// Convert to forward slashes and remove extension
	relPath = filepath.ToSlash(relPath)
	importPath := strings.TrimSuffix(relPath, filepath.Ext(relPath))

	return importPath
}

func (c *CompilerContext) FullPathToModuleName(fullPath string) string {
	relPath, err := filepath.Rel(c.ProjectRoot, fullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return ""
	}
	filename := filepath.Base(fullPath)
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// IsRemoteImport checks if an import path is a remote module (github.com/*, gitlab.com/*, etc.)
func (c *CompilerContext) IsRemoteImport(importPath string) bool {
	return strings.HasPrefix(importPath, "github.com/") ||
		strings.HasPrefix(importPath, "gitlab.com/") ||
		strings.HasPrefix(importPath, "bitbucket.org/")
}

// ParseRemoteImport parses a remote import path and extracts version information
// Returns: modulePath, version, subpath
// Example: "github.com/user/repo/folder/mod@v1.0.0" -> "github.com/user/repo", "v1.0.0", "folder/mod"
func (c *CompilerContext) ParseRemoteImport(importPath string) (string, string, string) {
	// Check for version specifier
	atIndex := strings.LastIndex(importPath, "@")
	var version string
	var pathWithoutVersion string

	if atIndex != -1 {
		version = importPath[atIndex+1:]
		pathWithoutVersion = importPath[:atIndex]
	} else {
		version = "latest"
		pathWithoutVersion = importPath
	}

	// Parse the path to extract repo and subpath
	parts := strings.Split(pathWithoutVersion, "/")
	if len(parts) < 3 {
		return "", "", ""
	}

	// For github.com/user/repo/folder/mod -> repo is "github.com/user/repo"
	repoPath := strings.Join(parts[:3], "/")
	var subPath string
	if len(parts) > 3 {
		subPath = strings.Join(parts[3:], "/")
	}

	return repoPath, version, subPath
}

// GetRemoteModuleCachePath returns the cache path for a remote module
func (c *CompilerContext) GetRemoteModuleCachePath(repoPath, version string) string {
	return filepath.Join(c.RemoteCachePath, repoPath+"@"+version)
}

// IsRemoteModuleCached checks if a remote module is already cached locally
func (c *CompilerContext) IsRemoteModuleCached(repoPath, version string) bool {
	cachePath := c.GetRemoteModuleCachePath(repoPath, version)
	_, err := os.Stat(cachePath)
	return err == nil
}

func (c *CompilerContext) GetModule(importPath string) (*Module, error) {
	if c.Modules == nil {
		return nil, fmt.Errorf("module '%s' not found in context", importPath)
	}
	module, exists := c.Modules[importPath]
	if !exists {
		return nil, fmt.Errorf("module '%s' not found in context", importPath)
	}
	return module, nil
}

func (c *CompilerContext) RemoveModule(importPath string) {
	if c.Modules == nil {
		return
	}
	if _, exists := c.Modules[importPath]; !exists {
		return
	}
	delete(c.Modules, importPath)
}

func (c *CompilerContext) ModuleCount() int {
	if c.Modules == nil {
		return 0
	}
	return len(c.Modules)
}

func (c *CompilerContext) PrintModules() {
	if c == nil {
		colors.YELLOW.Println("No modules in cache (context is nil)")
		return
	}
	modules := c.ModuleNames()
	if len(modules) == 0 {
		colors.YELLOW.Println("No modules in cache")
		return
	}

	//sort
	sort.Strings(modules)

	colors.BLUE.Println("Modules in cache:")
	for _, name := range modules {
		module, exists := c.Modules[name]
		if exists && module.IsBuiltin {
			colors.PURPLE.Printf("- %s ", name)
			colors.LIGHT_BLUE.Println("(built-in)")
		} else {
			colors.PURPLE.Printf("- %s\n", name)
		}
	}
}

func (c *CompilerContext) ModuleNames() []string {
	if c.Modules == nil {
		return []string{}
	}
	names := make([]string, 0, len(c.Modules))
	for key := range c.Modules {
		names = append(names, key)
	}
	return names
}

func (c *CompilerContext) HasModule(importPath string) bool {
	if c.Modules == nil {
		return false
	}
	_, exists := c.Modules[importPath]
	return exists
}

// IsModuleParsed checks if a module has been parsed (at least PhaseParsed)
func (c *CompilerContext) IsModuleParsed(importPath string) bool {
	if c.Modules == nil {
		return false
	}
	module, exists := c.Modules[importPath]
	return exists && module.Phase >= PhaseParsed
}

// GetModulePhase returns the current processing phase of a module
func (c *CompilerContext) GetModulePhase(importPath string) ModulePhase {
	if c.Modules == nil {
		return PhaseNotStarted
	}
	module, exists := c.Modules[importPath]
	if !exists {
		return PhaseNotStarted
	}
	return module.Phase
}

// SetModulePhase updates the processing phase of a module
func (c *CompilerContext) SetModulePhase(importPath string, phase ModulePhase) {
	if c.Modules == nil {
		return
	}
	module, exists := c.Modules[importPath]
	if !exists {
		return
	}
	module.Phase = phase
}

// CanProcessPhase checks if a module is ready for a specific phase
func (c *CompilerContext) CanProcessPhase(importPath string, requiredPhase ModulePhase) bool {
	currentPhase := c.GetModulePhase(importPath)
	// Can only process the next phase in sequence
	return currentPhase == requiredPhase-1
}

func (c *CompilerContext) AddModule(importPath string, module *ast.Program, isBuiltin bool) {
	if c.Modules == nil {
		c.Modules = make(map[string]*Module)
	}
	if _, exists := c.Modules[importPath]; exists {
		return
	}
	if module == nil {
		panic(fmt.Sprintf("Cannot add nil module for '%s'\n", importPath))
	}
	c.Modules[importPath] = &Module{
		AST:         module,
		SymbolTable: NewSymbolTable(c.Builtins),
		Phase:       PhaseParsed, // Module is parsed when added
		IsBuiltin:   isBuiltin,
	}
}

// isModuleParsing checks if a module is currently being parsed
func (c *CompilerContext) isModuleParsing(importPath string) bool {
	if c._parsingModules == nil {
		return false
	}
	return c._parsingModules[importPath]
}

// DetectCycle detects if adding an edge from 'from' to 'to' would create a cycle
// Returns the cycle path starting from the original module if a cycle is detected
func (c *CompilerContext) DetectCycle(from, to string) ([]string, bool) {
	// Normalize paths to handle forward/backward slash inconsistency
	from = filepath.ToSlash(from)
	to = filepath.ToSlash(to)

	colors.CYAN.Printf("DetectCycle: %s → %s\n", filepath.Base(from), filepath.Base(to))

	// Initialize DepGraph if needed
	if c.DepGraph == nil {
		c.DepGraph = make(map[string][]string)
	}

	// Check if this edge would create a cycle by doing a DFS from 'to' to see if we can reach 'from'
	visited := make(map[string]bool)
	path := make([]string, 0)

	if cycle := c.findCyclePath(to, from, visited, path); cycle != nil {
		// Found a cycle, return it WITHOUT adding the edge
		colors.RED.Printf("CYCLE DETECTED: %v\n", cycle)
		return cycle, true
	}

	// No cycle found, add the edge (with normalized paths)
	c.DepGraph[from] = append(c.DepGraph[from], to)
	colors.GREEN.Printf("Edge added: %s → %s\n", filepath.Base(from), filepath.Base(to))
	return nil, false
} // findCyclePath uses DFS to find if there's a path from 'start' to 'target'
// If found, returns the complete cycle path
func (c *CompilerContext) findCyclePath(start, target string, visited map[string]bool, path []string) []string {
	// Normalize paths
	start = filepath.ToSlash(start)
	target = filepath.ToSlash(target)

	if start == target {
		// Found the target, construct the cycle
		cyclePath := make([]string, len(path)+2)
		cyclePath[0] = target // Start the cycle from target
		copy(cyclePath[1:], path)
		cyclePath[len(cyclePath)-1] = target // Close the cycle
		return cyclePath
	}

	if visited[start] {
		return nil // Already visited this node
	}

	visited[start] = true
	path = append(path, start)

	// Visit all neighbors
	for _, neighbor := range c.DepGraph[start] {
		neighbor = filepath.ToSlash(neighbor) // Normalize neighbor path
		if cycle := c.findCyclePath(neighbor, target, visited, path); cycle != nil {
			return cycle
		}
	}

	return nil
}

// StartParsing marks a module as currently being parsed
func (c *CompilerContext) StartParsing(importPath string) {
	if c._parsingModules == nil {
		c._parsingModules = make(map[string]bool)
	}
	if c._parsingStack == nil {
		c._parsingStack = make([]string, 0)
	}

	c._parsingModules[importPath] = true
	c._parsingStack = append(c._parsingStack, importPath)
}

// FinishParsing marks a module as no longer being parsed
func (c *CompilerContext) FinishParsing(importPath string) {
	if c._parsingModules != nil {
		delete(c._parsingModules, importPath)
	}

	// Remove from stack (should be the last element)
	if len(c._parsingStack) > 0 && c._parsingStack[len(c._parsingStack)-1] == importPath {
		c._parsingStack = c._parsingStack[:len(c._parsingStack)-1]
	}
}

// getModulesPath determines the path to the system built-in modules
// It looks for a 'modules' directory relative to the compiler binary location
func getModulesPath() string {
	// Get the executable path
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to current working directory if we can't get executable path
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, "modules")
	}

	// Get the directory containing the executable
	execDir := filepath.Dir(execPath)

	// Look for modules directory relative to the executable
	// This handles both development and production scenarios
	modulesPath := filepath.Join(execDir, "..", "modules")

	// Check if the modules directory exists
	if _, err := os.Stat(modulesPath); err == nil {
		absPath, _ := filepath.Abs(modulesPath)
		return filepath.ToSlash(absPath)
	}

	// Fallback: look in the same directory as the executable
	modulesPath = filepath.Join(execDir, "modules")
	if _, err := os.Stat(modulesPath); err == nil {
		absPath, _ := filepath.Abs(modulesPath)
		return filepath.ToSlash(absPath)
	}

	// Last resort: return the expected path even if it doesn't exist
	// This allows for future module installation
	absPath, _ := filepath.Abs(filepath.Join(execDir, "..", "modules"))
	return filepath.ToSlash(absPath)
}

func NewCompilerContext(entrypointFullpath string) *CompilerContext {
	if contextCreated {
		panic("CompilerContext already created, cannot create a new one")
	}
	contextCreated = true

	// Load project configuration
	root, err := config.FindProjectRoot(entrypointFullpath)
	if err != nil {
		panic(err)
	}

	projectConfig, err := config.LoadProjectConfig(root)
	if err != nil {
		colors.RED.Printf("Failed to load project config: %s\n", err)
		os.Exit(1)
	}

	if err = config.ValidateProjectConfig(projectConfig); err != nil {
		colors.RED.Printf("Invalid project configuration: %s\n", err)
		os.Exit(1)
	}

	//get the entry point relative to the project root
	entryPoint, err := filepath.Rel(root, entrypointFullpath)
	if err != nil {
		colors.RED.Printf("Failed to get relative path for entry point: %s\n", err)
		os.Exit(1)
	}
	entryPoint = filepath.ToSlash(entryPoint) // Ensure forward slashes for consistency

	// Determine modules path relative to compiler binary
	modulesPath := getModulesPath()

	// Set up remote module cache path
	remoteCachePath := filepath.Join(root, ".ferret", "modules")
	remoteCachePath = filepath.ToSlash(remoteCachePath)
	os.MkdirAll(remoteCachePath, 0755)

	// Debug: print modules path for troubleshooting
	if len(os.Args) > 1 && strings.Contains(strings.Join(os.Args, " "), "--debug") {
		colors.YELLOW.Printf("Modules path: %s\n", modulesPath)
		colors.YELLOW.Printf("Remote cache path: %s\n", remoteCachePath)
	}

	return &CompilerContext{
		EntryPoint:      entryPoint,
		Builtins:        AddPreludeSymbols(NewSymbolTable(nil)), // Initialize built-in symbols
		Modules:         make(map[string]*Module),
		Reports:         report.Reports{},
		ProjectConfig:   projectConfig,
		ProjectRoot:     root,
		ModulesPath:     modulesPath,
		RemoteCachePath: remoteCachePath,
	}
}

func (c *CompilerContext) Destroy() {
	if !contextCreated {
		return
	}
	contextCreated = false

	c.Modules = nil
	c.Reports = nil
	c.DepGraph = nil
}
