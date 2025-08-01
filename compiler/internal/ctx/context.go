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
	"compiler/internal/modules"
	"compiler/internal/report"
	"compiler/internal/symbol"
)

var contextCreated = false

type CompilerContext struct {
	EntryPoint string                     // Entry point file
	Builtins   *symbol.SymbolTable        // Built-in symbols, e.g., "i32", "f64", "str", etc.
	Modules    map[string]*modules.Module // key: import path
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
	if c.IsBuiltinModuleFile(fullPath) {
		return c.getBuiltinModuleImportPath(fullPath)
	}

	// Check if this is a remote module file
	if c.IsRemoteModuleFile(fullPath) {
		return c.CachePathToImportPath(fullPath)
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

// IsRemoteModuleFile checks if the given file path is within the remote modules cache
func (c *CompilerContext) IsRemoteModuleFile(fullPath string) bool {
	if c.RemoteCachePath == "" {
		return false
	}
	absRemotePath, err := filepath.Abs(c.RemoteCachePath)
	if err != nil {
		return false
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absFullPath, absRemotePath)
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

func (c *CompilerContext) FullPathToModuleName(fullPath string) string {
	// Removed: For modules outside the project root, like built-in modules, it cannot match with the project root.
	// relPath, err := filepath.Rel(c.ProjectRoot, fullPath)
	// if err != nil || strings.HasPrefix(relPath, "..") {
	// 	return ""
	// }
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
// Example: "github.com/user/repo@v1.0.0/data/types" -> "github.com/user/repo", "v1.0.0", "data/types"
func (c *CompilerContext) ParseRemoteImport(importPath string) (string, string, string) {
	// Check for version specifier using @
	atIndex := strings.Index(importPath, "@")
	var version string
	var pathWithoutVersion string

	if atIndex != -1 {
		// Split at @ to get the part before and after
		beforeAt := importPath[:atIndex]
		afterAt := importPath[atIndex+1:]

		// Check if this looks like a repo@version pattern
		parts := strings.Split(beforeAt, "/")
		if len(parts) >= 3 {
			// This is likely github.com/user/repo@version
			pathWithoutVersion = beforeAt

			// Find where version ends (at next / or end of string)
			slashIndex := strings.Index(afterAt, "/")
			if slashIndex != -1 {
				version = afterAt[:slashIndex]
				// The rest after version is subpath, prepend to pathWithoutVersion
				subpathAfterVersion := afterAt[slashIndex+1:]
				pathWithoutVersion = beforeAt + "/" + subpathAfterVersion
			} else {
				version = afterAt
			}
		} else {
			// @ is not in the expected repo position, treat as no version
			version = "latest"
			pathWithoutVersion = importPath
		}
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

// IsRemoteModuleCached checks if a remote module is cached using flat structure
// flatModuleName format: "github.com/user/repo@version"
func (c *CompilerContext) IsRemoteModuleCached(flatModuleName string) bool {
	// Return false for empty module names
	if flatModuleName == "" {
		return false
	}

	cachePath := filepath.Join(c.RemoteCachePath, flatModuleName)
	_, err := os.Stat(cachePath)
	return err == nil
}

func (c *CompilerContext) GetModule(importPath string) (*modules.Module, error) {
	if c.Modules == nil {
		return nil, fmt.Errorf("module '%s' not found in context", importPath)
	}
	module, exists := c.Modules[importPath]
	if !exists {
		return nil, fmt.Errorf("module '%s' not found in context", importPath)
	}

	// ✅ SECURITY CHECK: For remote modules, verify share setting every time they're accessed
	if strings.HasPrefix(importPath, "github.com/") {
		if err := c.validateRemoteModuleShareSetting(importPath); err != nil {
			return nil, err
		}
	}

	return module, nil
}

// validateRemoteModuleShareSetting checks if a remote module allows sharing
// This is called every time a remote module is accessed from cache
func (c *CompilerContext) validateRemoteModuleShareSetting(importPath string) error {
	// Extract repo name from import path
	parts := strings.Split(importPath, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid remote import path format: %s", importPath)
	}

	// Load lockfile to get dependency information
	lockfile, err := modules.LoadLockfile(c.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Get repo path (github.com/user/repo)
	repoPath := strings.Join(parts[:3], "/")
	var version string
	found := false
	for key := range lockfile.Dependencies {
		if strings.HasPrefix(key, repoPath+"@") {
			version = lockfile.Dependencies[key].Version
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("remote module %s not found in lockfile", repoPath)
	}

	// Build cache directory path
	repoName := strings.Join(parts[1:3], "/") // user/repo
	moduleDir := filepath.Join(c.RemoteCachePath, "github.com", repoName+"@"+version)

	// Get the specific module file path for project-level checking
	var moduleFilePath string
	if len(parts) > 3 {
		// Has sub-path like github.com/user/repo/data/bigint
		modulePath := strings.Join(parts[3:], "/")
		moduleFilePath = filepath.Join(moduleDir, modulePath+".fer")
	} else {
		// Just the repo like github.com/user/repo
		moduleFilePath = filepath.Join(moduleDir, "fer.ret")
	}

	// Use the modules package function for consistency
	canShare, err := modules.CheckRemoteModuleShareSetting(moduleFilePath)
	if err != nil {
		return fmt.Errorf("failed to check share settings for module %s: %w", repoPath, err)
	}

	if !canShare {
		return fmt.Errorf("module %s has disabled sharing (share = false). Cannot use this module", repoPath)
	}

	return nil
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
		module, exists := c.Modules[name]
		if exists {
			colors.PURPLE.Printf("- %s ", name)

			// Color-code by module type
			switch module.Type {
			case modules.LOCAL:
				colors.LIGHT_BLUE.Printf("(%s)", module.Type)
			case modules.REMOTE:
				colors.GREEN.Printf("(%s)", module.Type)
			case modules.BUILTIN:
				colors.YELLOW.Printf("(%s)", module.Type)
			default:
				colors.WHITE.Printf("(%s)", module.Type)
			}

			fmt.Println() // New line
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

func (c *CompilerContext) AddModule(importPath string, module *ast.Program, isBuiltin bool) {
	if c.Modules == nil {
		c.Modules = make(map[string]*modules.Module)
	}
	if _, exists := c.Modules[importPath]; exists {
		return
	}
	if module == nil {
		panic(fmt.Sprintf("Cannot add nil module for '%s'\n", importPath))
	}

	c.Modules[importPath] = &modules.Module{
		AST:            module,
		SymbolTable:    symbol.NewSymbolTable(c.Builtins),
		FunctionScopes: make(map[string]*symbol.SymbolTable),
		Phase:          modules.PHASE_PARSED, // Module is parsed when added
		IsBuiltin:      isBuiltin,
		Type:           modules.GetModuleType(importPath, c.ProjectConfig.Name),
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
		colors.RED.Printf("CYCLE DETECTED: %v\n", cycle)
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
		Builtins:        symbol.AddPreludeSymbols(symbol.NewSymbolTable(nil)), // Initialize built-in symbols
		Modules:         make(map[string]*modules.Module),
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
