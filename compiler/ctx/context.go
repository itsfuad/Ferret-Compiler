package ctx

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/utils/path"
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
	RootDir     string             // Root directory of the project
	EntryPoint  string             // Entry point file
	Modules     map[string]*Module // key: ModuleKey.String()
	Reports     report.Reports
	CachePath   string
	AliasToPath map[string]string // import alias -> file path
	// Dependency graph: key is importer, value is list of imported module keys (as strings)
	DepGraph map[string][]string
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
	colors.GREEN.Printf("Adding module: Key: %s, FilePath: %s\n", moduleName, module.FilePath)
	if c.Modules == nil {
		c.Modules = make(map[string]*Module)
	}
	if _, exists := c.Modules[moduleName]; exists {
		return
	}
	if module == nil {
		panic(fmt.Sprintf("Cannot add nil module for '%s'\n", moduleName))
	}
	c.Modules[moduleName] = &Module{AST: module, SymbolTable: semantic.NewSymbolTable(nil)}
}

func NewCompilerContext(entrypointPath string) *CompilerContext {
	if contextCreated {
		panic("CompilerContext already created, cannot create a new one")
	}
	contextCreated = true

	entrypointPath = path.ToAbs(entrypointPath)

	// Set root directory to the parent of the entry point's directory
	// This ensures imports like "code/maths/symbols/pi" resolve correctly from project root
	rootDir := filepath.Dir(entrypointPath)
	entryPoint := filepath.Base(entrypointPath)

	colors.ORANGE.Printf("Root dir: %s\n", rootDir)
	colors.ORANGE.Printf("Entry point: %s\n", entryPoint)

	cachePath := filepath.Join(rootDir, ".ferret", "modules")
	os.MkdirAll(cachePath, 0755)
	return &CompilerContext{
		RootDir:     rootDir,
		EntryPoint:  entryPoint,
		Modules:     make(map[string]*Module),
		Reports:     report.Reports{},
		AliasToPath: make(map[string]string),
		CachePath:   cachePath,
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

func (c *CompilerContext) AbsToModuleName(absPath string) string {
	relPath, err := filepath.Rel(c.RootDir, absPath)
	if err != nil {
		panic(err)
	}
	moduleName := filepath.Base(relPath)
	moduleName = strings.TrimSuffix(moduleName, filepath.Ext(moduleName))
	return moduleName
}