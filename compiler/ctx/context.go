package ctx

import (
	"compiler/colors"
	"compiler/internal/config"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var contextCreated = false

type Module struct {
	AST         *ast.Program
	SymbolTable *semantic.SymbolTable
}

type CompilerContext struct {
	EntryPoint        string                // Entry point file
	Builtins          *semantic.SymbolTable // Built-in symbols, e.g., "i32", "f64", "str", etc.
	Modules           map[string]*Module    // key: ModuleKey.String()
	Reports           report.Reports
	CachePath         string
	AliasToModuleName map[string]string // import alias -> file path
	// Dependency graph: key is importer, value is list of imported module keys (as strings)
	DepGraph map[string][]string
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
	fmt.Printf("Logical Path: %s\n", logicalPath)
	for key, value := range c.RemoteConfigs {
		fmt.Printf("Key: %s, Value: %s\n", key, value)
	}

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

func (c *CompilerContext) GetModule(key string) *Module {
	if c.Modules == nil {
		return nil
	}
	module, exists := c.Modules[key]
	if !exists {
		return nil
	}
	return module
}

func (c *CompilerContext) RemoveModule(key string) {
	if c.Modules == nil {
		return
	}
	if _, exists := c.Modules[key]; !exists {
		return
	}
	delete(c.Modules, key)
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
		colors.GREEN.Printf("- %s\n", name)
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

func (c *CompilerContext) HasModule(moduleName string) bool {
	if c.Modules == nil {
		return false
	}
	_, exists := c.Modules[moduleName]
	return exists
}

func (c *CompilerContext) AddModule(moduleName string, module *ast.Program) {
	colors.GREEN.Printf("Adding module: Key: %s, FilePath: %s\n", moduleName, module.FullPath)
	if c.Modules == nil {
		c.Modules = make(map[string]*Module)
	}
	if _, exists := c.Modules[moduleName]; exists {
		return
	}
	if module == nil {
		panic(fmt.Sprintf("Cannot add nil module for '%s'\n", moduleName))
	}
	c.Modules[moduleName] = &Module{AST: module, SymbolTable: semantic.NewSymbolTable(c.Builtins)}
}

func NewCompilerContext(entrypointPath string) *CompilerContext {
	if contextCreated {
		panic("CompilerContext already created, cannot create a new one")
	}
	contextCreated = true

	entrypointPath, err := filepath.Abs(entrypointPath)
	if err != nil {
		panic(fmt.Errorf("failed to get full path: %w", err))
	}
	entrypointPath = filepath.ToSlash(entrypointPath)

	// Load project configuration
	projectConfig, projectRoot, err := config.LoadProjectConfig(filepath.Dir(entrypointPath))
	if err != nil {
		// If no project config found, create a default one
		colors.YELLOW.Printf("No ferret.project.json found, creating default configuration\n")
		projectRoot = filepath.Dir(entrypointPath)
		if err := config.CreateDefaultProjectConfig(projectRoot); err != nil {
			panic(fmt.Errorf("failed to create default project config: %w", err))
		}
		projectConfig, _, err = config.LoadProjectConfig(projectRoot)
		if err != nil {
			panic(fmt.Errorf("failed to load default project config: %w", err))
		}
	}

	// Set root directory to the project root
	entryPoint := filepath.Base(entrypointPath)
	entryPoint = filepath.ToSlash(entryPoint)

	colors.ORANGE.Printf("Project root: %s\n", projectRoot)
	colors.ORANGE.Printf("Entry point: %s\n", entryPoint)

	// Use cache path from project config
	cachePath := filepath.Join(projectRoot, projectConfig.Cache.Path)
	cachePath = filepath.ToSlash(cachePath)
	os.MkdirAll(cachePath, 0755)

	return &CompilerContext{
		EntryPoint:        entryPoint,
		Builtins:          semantic.AddPreludeSymbols(semantic.NewSymbolTable(nil)), // Initialize built-in symbols
		Modules:           make(map[string]*Module),
		Reports:           report.Reports{},
		AliasToModuleName: make(map[string]string),
		CachePath:         cachePath,
		ProjectConfig:     projectConfig,
		ProjectRoot:       projectRoot,
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
	c.DepGraph[importer] = append(c.DepGraph[importer], imported)
}

// DetectCycle checks for a cycle starting from the given module key string, returns the cycle path if found
func (c *CompilerContext) DetectCycle(start string) ([]string, bool) {
	visited := make(map[string]bool)
	stack := make([]string, 0)
	var dfs func(node string) ([]string, bool)
	dfs = func(node string) ([]string, bool) {
		if visited[node] {
			for i, n := range stack {
				if n == node {
					return append(stack[i:], node), true
				}
			}
			return nil, false
		}
		visited[node] = true
		stack = append(stack, node)
		for _, neighbor := range c.DepGraph[node] {
			if path, found := dfs(neighbor); found {
				return path, true
			}
		}
		stack = stack[:len(stack)-1]
		visited[node] = false
		return nil, false
	}
	return dfs(start)
}

func (c *CompilerContext) FullPathToModuleName(fullPath string) string {
	relPath, err := filepath.Rel(c.ProjectRoot, fullPath)
	if err != nil {
		return fullPath // Fallback to full path if relative path cannot be determined
	}
	relPath = filepath.ToSlash(relPath)
	moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	return moduleName
}