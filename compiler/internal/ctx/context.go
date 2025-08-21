package ctx

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"compiler/colors"
	"compiler/config"
	"compiler/constants"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/symbol"
	"compiler/internal/utils/fs"
	"compiler/internal/utils/stack"
	"compiler/report"
)

var contextCreated = false

const BUILTIN_DIR = "../modules"

type CompilerContext struct {
	EntryPoint string                     // Entry point file
	Builtins   *symbol.SymbolTable        // Built-in symbols, e.g., "i32", "f64", "str", etc.
	Modules    map[string]*modules.Module // key: import path
	Reports    report.Reports
	// Project configuration
	ProjectConfig       *config.ProjectConfig
	ProjectStack        *stack.Stack[*config.ProjectConfig] // Stack of project configurations for nested imports
	ProjectRootFullPath string
	BuiltinModules      map[string]string // key: projectname, value: path

	// Remote module cache path (.ferret)
	RemoteCachePath string

	// Dependency graph: key is importer, value is list of imported module keys (as strings)
	DepGraph map[string][]string
}

// ResolveImportPath categorizes an import path with proper neighbor package name resolution
func (c *CompilerContext) ResolveImportPath(importPath string) (*config.ProjectConfig, string, modules.ModuleType, error) {
	if importPath == "" {
		return nil, "", modules.UNKNOWN, fmt.Errorf("empty import path")
	}

	packageName := fs.FirstPart(importPath)
	currentProjectConfig := c.ProjectStack.Peek()
	cleanPath := strings.TrimPrefix(importPath, packageName+"/")

	// Determine module type and resolve path
	projectConfig, resolvedPath, modType, err := c.resolveByType(packageName, cleanPath, importPath, currentProjectConfig)
	if err != nil {
		return nil, "", modType, err
	}

	// Validate the resolved path
	finalPath, err := c.validateResolvedPath(resolvedPath, importPath)
	if err != nil {
		return nil, "", modType, err
	}

	return projectConfig, finalPath, modType, nil
}

// resolveByType determines the module type and resolves the path accordingly
func (c *CompilerContext) resolveByType(packageName, cleanPath, importPath string, currentProjectConfig *config.ProjectConfig) (*config.ProjectConfig, string, modules.ModuleType, error) {
	// Local module
	if packageName == currentProjectConfig.Name {
		resolvedPath := filepath.Join(currentProjectConfig.ProjectRoot, cleanPath)
		return currentProjectConfig, resolvedPath, modules.LOCAL, nil
	}

	// Remote module
	if c.IsRemoteImport(importPath) {
		return c.resolveRemoteModule(importPath, currentProjectConfig)
	}

	// Neighbor module
	if rel, found := currentProjectConfig.Neighbors.Projects[packageName]; found {
		return c.resolveNeighborModule(rel, cleanPath, currentProjectConfig)
	}

	// Builtin module
	if path, found := c.BuiltinModules[packageName]; found {
		return c.resolveBuiltinModule(path, cleanPath, packageName)
	}

	// Unknown module type
	return nil, "", modules.UNKNOWN, nil
}

// resolveRemoteModule handles remote module resolution
func (c *CompilerContext) resolveRemoteModule(importPath string, currentProjectConfig *config.ProjectConfig) (*config.ProjectConfig, string, modules.ModuleType, error) {

	repoPath, err := modules.ExtractRepoPathFromImport(importPath)
	if err != nil {
		return nil, "", modules.REMOTE, err
	}
	// Check if module is in dependencies
	version, found := currentProjectConfig.Dependencies.Packages[repoPath]
	if !found {
		return nil, "", modules.REMOTE, fmt.Errorf("remote module %q is not installed\nRun `ferret get %s` to install it", repoPath, repoPath)
	}

	host, owner, repo, _, err := modules.SplitRepo(repoPath)
	if err != nil {
		return nil, "", modules.REMOTE, err
	}

	// Check if module is cached
	if !modules.IsModuleCached(c.RemoteCachePath, repoPath, version) {
		return nil, "", modules.REMOTE, fmt.Errorf("module %q is not cached\nRun `ferret get` to install it", repoPath)
	}

	modulename, err := modules.ExtractModuleFromImport(importPath)
	if err != nil {
		return nil, "", modules.REMOTE, err
	}

	resolvedPath := filepath.Join(c.RemoteCachePath, host, owner, modules.BuildPackageSpec(repo, version), modulename)

	return currentProjectConfig, resolvedPath, modules.REMOTE, nil
}

// resolveNeighborModule handles neighbor module resolution
func (c *CompilerContext) resolveNeighborModule(rel, cleanPath string, currentProjectConfig *config.ProjectConfig) (*config.ProjectConfig, string, modules.ModuleType, error) {
	neighborProject, err := config.LoadProjectConfig(filepath.Join(currentProjectConfig.ProjectRoot, rel))
	if err != nil {
		return nil, "", modules.NEIGHBOR, err
	}

	resolvedPath := filepath.Join(neighborProject.ProjectRoot, cleanPath)
	return neighborProject, resolvedPath, modules.NEIGHBOR, nil
}

// resolveBuiltinModule handles builtin module resolution
func (c *CompilerContext) resolveBuiltinModule(path, cleanPath, packageName string) (*config.ProjectConfig, string, modules.ModuleType, error) {
	builtinProject, err := config.LoadProjectConfig(path)
	if err != nil {
		return nil, "", modules.BUILTIN, err
	}

	resolvedPath := filepath.Join(builtinProject.ProjectRoot, cleanPath)
	return builtinProject, resolvedPath, modules.BUILTIN, nil
}

// validateResolvedPath validates the final resolved path
func (c *CompilerContext) validateResolvedPath(resolvedPath, importPath string) (string, error) {
	// Check if it's a directory (not allowed)
	if fs.IsDir(resolvedPath) {
		return "", fmt.Errorf("%q is not a module", importPath)
	}

	// Add extension and check if file exists
	finalPath := resolvedPath + constants.EXT
	if !fs.IsValidFile(finalPath) {
		fmt.Printf("Final path: %s\n", finalPath)
		return "", fmt.Errorf("module %q does not exist", importPath)
	}

	return finalPath, nil
}


// CachePathToImportPath converts a remote module file path back to its import path
func (c *CompilerContext) CachePathToImportPath(fullPath string) string {
	// Convert: D:\...\cache\github.com\itsfuad\ferret-mod\data\bigint.fer
	// To: github.com/itsfuad/ferret-mod/data/bigint

	absRemotePath, err := filepath.Abs(c.RemoteCachePath)
	if err != nil {
		return ""
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return ""
	}

	// Get relative path within cache
	relPath, err := filepath.Rel(absRemotePath, absFullPath)
	if err != nil {
		return ""
	}

	// Normalize to forward slashes
	relPath = filepath.ToSlash(relPath)

	// Remove file extension
	relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))

	// Since we now store with full github.com path, we can use it directly
	// Just remove any version suffixes from parts
	parts := strings.Split(relPath, "/")
	var result []string
	for _, part := range parts {
		if strings.Contains(part, "@") {
			// Remove version from repo name
			repoName := strings.Split(part, "@")[0]
			result = append(result, repoName)
		} else {
			result = append(result, part)
		}
	}

	return strings.Join(result, "/")
}

func (c *CompilerContext) FullPathToAlias(fullPath string) string {
	filename := filepath.Base(fullPath)
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// IsRemoteImport checks if an import path is a remote module (github.com/*, gitlab.com/*, etc.)
func (c *CompilerContext) IsRemoteImport(importPath string) bool {
	return strings.HasPrefix(importPath, "github.com/") ||
		strings.HasPrefix(importPath, "gitlab.com/") ||
		strings.HasPrefix(importPath, "bitbucket.org/")
}

func (c *CompilerContext) GetModule(importPath string) (*modules.Module, error) {
	if c.Modules == nil {
		return nil, fmt.Errorf("module context is empty")
	}
	module, exists := c.Modules[importPath]
	if !exists {
		return nil, fmt.Errorf("module %q not found in context: %#v", importPath, c.Modules)
	}

	return module, nil
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
	modulesStr := c.ModuleNames()
	if len(modulesStr) == 0 {
		colors.YELLOW.Println("No modules in cache")
		return
	}

	//sort
	sort.Strings(modulesStr)

	colors.BLUE.Println("Modules in cache:")
	for _, name := range modulesStr {
		_, exists := c.Modules[name]
		if exists {
			colors.PURPLE.Printf("- %s ", name)
			fmt.Println() // New line
		} else {
			colors.PURPLE.Printf("- %s\n", name)
		}
	}
	//show the project entrypoint
	colors.CYAN.Printf("Project Entry Point: %s\n", c.EntryPoint)
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
	return exists && module.Phase >= modules.PHASE_PARSED
}

// GetModulePhase returns the current processing phase of a module
func (c *CompilerContext) GetModulePhase(importPath string) modules.ModulePhase {
	if c.Modules == nil {
		return modules.PHASE_NOT_STARTED
	}
	module, exists := c.Modules[importPath]
	if !exists {
		return modules.PHASE_NOT_STARTED
	}
	return module.Phase
}

// SetModulePhase updates the processing phase of a module
func (c *CompilerContext) SetModulePhase(importPath string, phase modules.ModulePhase) {
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
func (c *CompilerContext) CanProcessPhase(importPath string, requiredPhase modules.ModulePhase) bool {
	currentPhase := c.GetModulePhase(importPath)
	// Can only process the next phase in sequence
	return currentPhase == requiredPhase-1
}

func (c *CompilerContext) AddModule(importPath string, module *ast.Program) {
	if c.Modules == nil {
		c.Modules = make(map[string]*modules.Module)
	}
	if _, exists := c.Modules[importPath]; exists {
		return
	}
	if module == nil {
		panic(fmt.Sprintf("Cannot add nil module for %q\n", importPath))
	}

	c.Modules[importPath] = &modules.Module{
		AST:         module,
		SymbolTable: symbol.NewSymbolTable(c.Builtins),
		Phase:       modules.PHASE_PARSED, // Module is parsed when added
	}
}

// DetectCycle detects if adding an edge from 'from' to 'to' would create a cycle
// Returns the cycle path starting from the original module if a cycle is detected
func (c *CompilerContext) DetectCycle(from, to string) ([]string, bool) {
	// Normalize paths to handle forward/backward slash inconsistency
	from = filepath.ToSlash(from)
	to = filepath.ToSlash(to)

	// Initialize DepGraph if needed
	if c.DepGraph == nil {
		c.DepGraph = make(map[string][]string)
	}

	// Check if this edge would create a cycle by doing a DFS from 'to' to see if we can reach 'from'
	visited := make(map[string]bool)
	path := make([]string, 0)

	if cycle := c.findCyclePath(to, from, visited, path); cycle != nil {
		// Found a cycle, return it WITHOUT adding the edge
		return cycle, true
	}

	// No cycle found, add the edge (with normalized paths)
	c.DepGraph[from] = append(c.DepGraph[from], to)

	return nil, false
}

// findCyclePath uses DFS to find if there's a path from 'start' to 'target'
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

func NewCompilerContext(projectConfig *config.ProjectConfig) *CompilerContext {
	if contextCreated {
		panic("CompilerContext already created, cannot create a new one")
	}
	contextCreated = true

	if err := config.ValidateProjectConfig(projectConfig); err != nil {
		colors.RED.Printf("Invalid project configuration: %s\n", err)
		os.Exit(1)
	}

	//join entry point with project root
	entryPoint := filepath.Join(projectConfig.ProjectRoot, projectConfig.Build.Entry)

	entryPoint = filepath.ToSlash(entryPoint) // Ensure forward slashes for consistency

	// Set up remote module cache path
	remoteCachePath := filepath.Join(projectConfig.ProjectRoot, constants.CACHE_DIR)
	remoteCachePath = filepath.ToSlash(remoteCachePath)
	os.MkdirAll(remoteCachePath, 0755)

	execPath, err := os.Executable()
	if err != nil {
		colors.RED.Printf("Error getting executable path: %s\n", err)
		os.Exit(1)
	}

	// Initialize built-in modules
	BuiltinModules, err := fs.DirectChilds(filepath.Join(filepath.Dir(execPath), BUILTIN_DIR))
	if err != nil {
		colors.RED.Printf("Error reading built-in modules: %s\n", err)
		os.Exit(1)
	}

	return &CompilerContext{
		EntryPoint:          entryPoint,
		Builtins:            symbol.AddPreludeSymbols(symbol.NewSymbolTable(nil)), // Initialize built-in symbols
		Modules:             make(map[string]*modules.Module),
		Reports:             report.Reports{},
		ProjectConfig:       projectConfig,
		ProjectStack:        stack.New[*config.ProjectConfig](),
		RemoteCachePath:     remoteCachePath,
		BuiltinModules:      BuiltinModules,
		ProjectRootFullPath: projectConfig.ProjectRoot,
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
	c.BuiltinModules = nil
	c.ProjectStack = nil
}
