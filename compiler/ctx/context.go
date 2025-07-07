package ctx

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"os"
	"path/filepath"
)

var contextCreated = false

// ModuleKey uniquely identifies a module, distinguishing local and remote modules
// For local: IsRemote = false, Path = project-relative path
// For remote: IsRemote = true, Path = full remote import path (e.g., github.com/user/repo/path/file)
type ModuleKey struct {
	IsRemote bool
	Path     string // project-relative or remote import path
}

func (k ModuleKey) String() string {
	return k.Path
}

type CompilerContext struct {
	RootDir        string                  // Root directory of the project
	EntryPoint     string                  // Entry point file
	ModuleASTCache map[string]*ast.Program // key: ModuleKey.String()
	Reports        report.Reports
	CachePath      string

	// Dependency graph: key is importer, value is list of imported module keys (as strings)
	DepGraph map[string][]string
}

// Helpers to create module keys
func LocalModuleKey(path string) ModuleKey {
	return ModuleKey{IsRemote: false, Path: path}
}
func RemoteModuleKey(path string) ModuleKey {
	return ModuleKey{IsRemote: true, Path: path}
}

func (c *CompilerContext) GetModule(key ModuleKey) *ast.Program {
	if c.ModuleASTCache == nil {
		return nil
	}
	module, exists := c.ModuleASTCache[key.String()]
	if !exists {
		return nil
	}
	return module
}

func (c *CompilerContext) RemoveModule(key ModuleKey) {
	if c.ModuleASTCache == nil {
		return
	}
	if _, exists := c.ModuleASTCache[key.String()]; !exists {
		return
	}
	delete(c.ModuleASTCache, key.String())
}

func (c *CompilerContext) ModuleCount() int {
	if c.ModuleASTCache == nil {
		return 0
	}
	return len(c.ModuleASTCache)
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
	if c.ModuleASTCache == nil {
		return []string{}
	}
	names := make([]string, 0, len(c.ModuleASTCache))
	for key := range c.ModuleASTCache {
		names = append(names, key)
	}
	return names
}

func (c *CompilerContext) HasModule(key ModuleKey) bool {
	if c.ModuleASTCache == nil {
		return false
	}
	_, exists := c.ModuleASTCache[key.String()]
	return exists
}

func (c *CompilerContext) AddModule(key ModuleKey, module *ast.Program) {
	if c.ModuleASTCache == nil {
		c.ModuleASTCache = make(map[string]*ast.Program)
	}
	if _, exists := c.ModuleASTCache[key.String()]; exists {
		return
	}
	if module == nil {
		colors.RED.Printf("Cannot add nil module for '%s'\n", key.String())
		c.Reports.Add("CompilerContext", nil, "Cannot add nil module", report.SEMANTIC_PHASE).SetLevel(report.SYNTAX_ERROR)
		return
	}
	c.ModuleASTCache[key.String()] = module
}

func NewCompilerContext(entrypointPath string) *CompilerContext {
	if contextCreated {
		panic("CompilerContext already created, cannot create a new one")
	}
	contextCreated = true

	// Set root directory to the parent of the entry point's directory
	// This ensures imports like "code/maths/symbols/pi" resolve correctly from project root
	entryDir := filepath.Dir(entrypointPath) // returns xxx
	rootDir := filepath.Dir(entryDir)        // returns yyy
	entryPoint := filepath.Base(entrypointPath)

	// Ensure the root directory is absolute
	// if !filepath.IsAbs(rootDir) {
	// 	rootDir, _ = filepath.Abs(rootDir)
	// }

	cachePath := filepath.Join(rootDir, ".ferret", "modules")
	os.MkdirAll(cachePath, 0755)
	return &CompilerContext{
		RootDir:        rootDir,
		EntryPoint:     entryPoint,
		ModuleASTCache: make(map[string]*ast.Program),
		Reports:        nil,
		CachePath:      cachePath,
	}
}

func (c *CompilerContext) Destroy() {
	if !contextCreated {
		return
	}
	contextCreated = false

	c.ModuleASTCache = nil
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
