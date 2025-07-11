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
	"compiler/internal/semantic"
)

var contextCreated = false

type Module struct {
	AST         *ast.Program
	SymbolTable *semantic.SymbolTable
}

type CompilerContext struct {
	EntryPoint string                // Entry point file
	Builtins   *semantic.SymbolTable // Built-in symbols, e.g., "i32", "f64", "str", etc.
	Modules    map[string]*Module    // key: import path
	Reports    report.Reports
	CachePath  string
	// Dependency graph: key is importer, value is list of imported module keys (as strings)
	DepGraph map[string][]string
	// Track modules that are currently being parsed to prevent infinite recursion
	ParsingModules map[string]bool
	// Keep track of the parsing stack to show cycle paths
	ParsingStack []string
	// Project configuration
	ProjectConfig *config.ProjectConfig
	ProjectRoot   string
	RemoteConfigs map[string]bool
}

func (c *CompilerContext) GetConfigFile(configFilepath string) *config.ProjectConfig {
	if c.RemoteConfigs == nil {
		return nil
	}
	_, exists := c.RemoteConfigs[configFilepath]
	if !exists {
		return nil
	}
	cacheFile, err := os.ReadFile(configFilepath)
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
	if c.RemoteConfigs == nil {
		c.RemoteConfigs = make(map[string]bool)
	}
	c.RemoteConfigs[configFilepath] = true
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

	if c.RemoteConfigs == nil {
		return nil
	}

	logicalPath = filepath.ToSlash(logicalPath)
	parts := strings.Split(logicalPath, "/")

	// Start from full path, walk up to github.com/user/repo
	for i := len(parts); i >= 3; i-- {
		prefix := strings.Join(parts[:i], "/")
		if _, exists := c.RemoteConfigs[prefix]; exists {
			data, err := os.ReadFile(prefix)
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

func (c *CompilerContext) GetModule(importPath string) *Module {
	if c.Modules == nil {
		return nil
	}
	module, exists := c.Modules[importPath]
	if !exists {
		return nil
	}
	return module
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
	c.Modules[importPath] = &Module{AST: module, SymbolTable: semantic.NewSymbolTable(c.Builtins)}
}

// IsModuleParsing checks if a module is currently being parsed
func (c *CompilerContext) IsModuleParsing(importPath string) bool {
	if c.ParsingModules == nil {
		return false
	}
	return c.ParsingModules[importPath]
}

// GetCyclePath returns the cycle path if the given module is already being parsed
// Returns the complete path from the entry point, showing the full import chain
func (c *CompilerContext) GetCyclePath(importPath string) ([]string, bool) {
	if !c.IsModuleParsing(importPath) {
		return nil, false
	}

	// Find the first occurrence of the module that creates the cycle
	cycleStartIndex := -1
	for i, stackModule := range c.ParsingStack {
		if stackModule == importPath {
			cycleStartIndex = i
			break
		}
	}

	if cycleStartIndex == -1 {
		// This shouldn't happen, but handle it gracefully
		return c.ParsingStack, true
	}

	// Return the current parsing stack which shows the complete path from entry point
	// to where the cycle would occur (the unambiguous cycle path)
	return c.ParsingStack, true
}

// StartParsing marks a module as currently being parsed
func (c *CompilerContext) StartParsing(importPath string) {
	if c.ParsingModules == nil {
		c.ParsingModules = make(map[string]bool)
	}
	if c.ParsingStack == nil {
		c.ParsingStack = make([]string, 0)
	}

	c.ParsingModules[importPath] = true
	c.ParsingStack = append(c.ParsingStack, importPath)
}

// FinishParsing marks a module as no longer being parsed
func (c *CompilerContext) FinishParsing(importPath string) {
	if c.ParsingModules != nil {
		delete(c.ParsingModules, importPath)
	}

	// Remove from stack (should be the last element)
	if len(c.ParsingStack) > 0 && c.ParsingStack[len(c.ParsingStack)-1] == importPath {
		c.ParsingStack = c.ParsingStack[:len(c.ParsingStack)-1]
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
		Builtins:      semantic.AddPreludeSymbols(semantic.NewSymbolTable(nil)), // Initialize built-in symbols
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

// AddDepEdge adds an edge from importer to imported in the dependency graph
func (c *CompilerContext) AddDepEdge(importer, imported string) {
	if c.DepGraph == nil {
		c.DepGraph = make(map[string][]string)
	}
	colors.CYAN.Printf("Adding dependency edge: %s -> %s\n", importer, imported)
	c.DepGraph[importer] = append(c.DepGraph[importer], imported)

	// Debug: print current dependency graph
	colors.YELLOW.Println("Current dependency graph:")
	for from, tos := range c.DepGraph {
		for _, to := range tos {
			colors.YELLOW.Printf("  %s -> %s\n", from, to)
		}
	}
}

// DetectCycle checks for a cycle starting from the given module key string, returns the cycle path if found
func (c *CompilerContext) DetectCycle(start string) ([]string, bool) {
	colors.BLUE.Printf("Starting cycle detection from: %s\n", start)

	state := make(map[string]int) // 0 = unvisited, 1 = visiting, 2 = visited
	var stack []string

	var visit func(string) ([]string, bool)

	visit = func(node string) ([]string, bool) {
		switch state[node] {
		case 1:
			// Cycle found
			colors.RED.Printf("CYCLE DETECTED! Node %s is already in the recursion stack\n", node)
			for i, n := range stack {
				if n == node {
					cycle := append(stack[i:], node)
					colors.RED.Printf("Cycle path: %v\n", cycle)
					return cycle, true
				}
			}
			return []string{node}, true

		case 2:
			// Already visited
			colors.GREEN.Printf("Node %s already processed, skipping\n", node)
			return nil, false
		}

		// Mark as visiting
		state[node] = 1
		stack = append(stack, node)
		colors.BLUE.Printf("Visiting node: %s (stack: %v)\n", node, stack)

		// Visit neighbors
		for _, neighbor := range c.DepGraph[node] {
			if cycle, found := visit(neighbor); found {
				return cycle, true
			}
		}

		// Done processing
		stack = stack[:len(stack)-1]
		state[node] = 2
		colors.GREEN.Printf("Completed processing node: %s\n", node)
		return nil, false
	}

	cycle, found := visit(start)
	colors.BLUE.Printf("Cycle detection result: found=%v, cycle=%v\n", found, cycle)
	return cycle, found
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
