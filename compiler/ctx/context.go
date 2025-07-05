package ctx

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"os"
	"path/filepath"
)

var contextCreated = false

type CompilerContext struct {
	RootDir        	string // Root directory of the project
	EntryPoint	 	string // Entry point file
	ModuleASTCache 	map[string]*ast.Program
	Reports     	report.Reports
	CachePath       string
}

func (c *CompilerContext) GetModule(name string) *ast.Program {
	if c.ModuleASTCache == nil {
		return nil
	}
	module, exists := c.ModuleASTCache[name]
	if !exists {
		return nil
	}
	return module
}

func (c *CompilerContext) RemoveModule(name string) {
	if c.ModuleASTCache == nil {
		return
	}
	if _, exists := c.ModuleASTCache[name]; !exists {
		return
	}
	delete(c.ModuleASTCache, name)
}

func (c *CompilerContext) HasModule(name string) bool {
	if c.ModuleASTCache == nil {
		return false
	}
	_, exists := c.ModuleASTCache[name]
	return exists
}

func (c *CompilerContext) AddModule(name string, module *ast.Program) {
	if c.ModuleASTCache == nil {
		c.ModuleASTCache = make(map[string]*ast.Program)
	}
	if _, exists := c.ModuleASTCache[name]; exists {
		colors.RED.Printf("Module '%s' already exists in cache, skipping addition\n", name)
		return
	}
	if module == nil {
		colors.RED.Printf("Cannot add nil module for '%s'\n", name)
		c.Reports.Add("CompilerContext", nil, "Cannot add nil module", report.SEMANTIC_PHASE).SetLevel(report.SYNTAX_ERROR)
		return
	}
	c.ModuleASTCache[name] = module
	colors.GREEN.Printf("Module '%s' added to cache\n", name)
}

func NewCompilerContext(entrypointPath string) *CompilerContext {
	if contextCreated {
		panic("CompilerContext already created, cannot create a new one")
	}
	contextCreated = true

	rootDir := filepath.Dir(entrypointPath)
	entryPoint := filepath.Base(entrypointPath)

	// Ensure the root directory is absolute
	// if !filepath.IsAbs(rootDir) {
	// 	rootDir, _ = filepath.Abs(rootDir)
	// }

	cachePath := filepath.Join(rootDir, ".ferret", "cache")
	os.MkdirAll(cachePath, 0755)
	return &CompilerContext{
		RootDir:		rootDir,
		EntryPoint:		entryPoint,
		ModuleASTCache: make(map[string]*ast.Program),
		Reports:     nil,
		CachePath:   cachePath,
	}
}
