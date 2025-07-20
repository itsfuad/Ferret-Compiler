package ctx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
}

type CompilerContext struct {
	EntryPoint string             // Entry point file
	Builtins   *SymbolTable       // Built-in symbols, e.g., "i32", "f64", "str", etc.
	Modules    map[string]*Module // key: import path
	Reports    report.Reports
	CachePath  string
	// Project configuration
	ProjectConfig *config.ProjectConfig
	ProjectRoot   string

	remoteConfigs map[string]bool

	// Dependency graph: key is importer, value is list of imported module keys (as strings)
	DepGraph map[string][]string

	// Track modules that are currently being parsed to prevent infinite recursion
	_parsingModules map[string]bool
	// Keep track of the parsing stack to show cycle paths
	_parsingStack []string
}

func (c *CompilerContext) FullPathToImportPath(fullPath string) string {
	relPath, err := filepath.Rel(c.ProjectRoot, fullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return ""
	}
	relPath = filepath.ToSlash(relPath)
	moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	rootName := filepath.Base(c.ProjectRoot)
	return rootName + "/" + moduleName
}

func (c *CompilerContext) FullPathToModuleName(fullPath string) string {
	relPath, err := filepath.Rel(c.ProjectRoot, fullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return ""
	}
	filename := filepath.Base(fullPath)
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

func (c *CompilerContext) GetConfigFile(configFilepath string) *config.ProjectConfig {
	if c.remoteConfigs == nil {
		return nil
	}
	_, exists := c.remoteConfigs[configFilepath]
	if !exists {
		return nil
	}
	cacheFile, err := os.ReadFile(filepath.FromSlash(configFilepath))
	if err != nil {
		return nil
	}
	var projectConfig config.ProjectConfig
	if err := json.Unmarshal(cacheFile, &projectConfig); err != nil {
		return nil
	}
	return &projectConfig
}

func (c *CompilerContext) SetRemoteConfig(configFilepath string, data []byte) error {
	if c.remoteConfigs == nil {
		c.remoteConfigs = make(map[string]bool)
	}
	c.remoteConfigs[configFilepath] = true
	err := os.MkdirAll(filepath.Dir(configFilepath), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(configFilepath, data, 0644)
	if err != nil {
		return err
	}
	colors.GREEN.Printf("Cached remote config for %s\n", configFilepath)
	return nil
}

func (c *CompilerContext) FindNearestRemoteConfig(logicalPath string) *config.ProjectConfig {

	logicalPath = filepath.ToSlash(logicalPath)

	if c.remoteConfigs == nil {
		return nil
	}

	logicalPath = filepath.ToSlash(logicalPath)
	parts := strings.Split(logicalPath, "/")

	// Start from full path, walk up to github.com/user/repo
	for i := len(parts); i >= 3; i-- {
		prefix := strings.Join(parts[:i], "/")
		if _, exists := c.remoteConfigs[prefix]; exists {
			data, err := os.ReadFile(filepath.FromSlash(prefix))
			if err != nil {
				continue
			}
			var cfg config.ProjectConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				continue
			}
			return &cfg
		}
	}
	return nil
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
	colors.BLUE.Println("Modules in cache:")
	for _, name := range modules {
		colors.PURPLE.Printf("- %s\n", name)
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

func (c *CompilerContext) AddModule(importPath string, module *ast.Program) {
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
		panic(fmt.Errorf("failed to load project config: %w", err))
	}

	//get the entry point relative to the project root
	entryPoint, err := filepath.Rel(root, entrypointFullpath)
	if err != nil {
		panic(fmt.Errorf("failed to get relative path for entry point: %w", err))
	}
	entryPoint = filepath.ToSlash(entryPoint) // Ensure forward slashes for consistency

	// Use cache path from project config
	cachePath := filepath.Join(root, projectConfig.Cache.Path)
	cachePath = filepath.ToSlash(cachePath)
	os.MkdirAll(cachePath, 0755)

	return &CompilerContext{
		EntryPoint:    entryPoint,
		Builtins:      AddPreludeSymbols(NewSymbolTable(nil)), // Initialize built-in symbols
		Modules:       make(map[string]*Module),
		Reports:       report.Reports{},
		CachePath:     cachePath,
		ProjectConfig: projectConfig,
		ProjectRoot:   root,
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

	// Optionally, clear the cache directory
	if c.CachePath != "" {
		os.RemoveAll(c.CachePath)
	}
}
