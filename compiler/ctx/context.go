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
// For local: Kind = "local", Path = project-relative path
// For remote: Kind = "remote", Path = full remote import path (e.g., github.com/user/repo/path/file)
type ModuleKey struct {
	Kind string // "local" or "remote"
	Path string // project-relative or remote import path
}

func (k ModuleKey) String() string {
	return k.Kind + ":" + k.Path
}

type CompilerContext struct {
	RootDir        string                  // Root directory of the project
	EntryPoint     string                  // Entry point file
	ModuleASTCache map[string]*ast.Program // key: ModuleKey.String()
	Reports        report.Reports
	CachePath      string
}

// Helpers to create module keys
func LocalModuleKey(projectRelative string) ModuleKey {
	return ModuleKey{Kind: "local", Path: projectRelative}
}
func RemoteModuleKey(importPath string) ModuleKey {
	return ModuleKey{Kind: "remote", Path: importPath}
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
		colors.RED.Printf("Module '%s' already exists in cache, skipping addition\n", key.String())
		return
	}
	if module == nil {
		colors.RED.Printf("Cannot add nil module for '%s'\n", key.String())
		c.Reports.Add("CompilerContext", nil, "Cannot add nil module", report.SEMANTIC_PHASE).SetLevel(report.SYNTAX_ERROR)
		return
	}
	c.ModuleASTCache[key.String()] = module
	colors.GREEN.Printf("Module '%s' added to cache\n", key.String())
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
		RootDir:        rootDir,
		EntryPoint:     entryPoint,
		ModuleASTCache: make(map[string]*ast.Program),
		Reports:        nil,
		CachePath:      cachePath,
	}
}
