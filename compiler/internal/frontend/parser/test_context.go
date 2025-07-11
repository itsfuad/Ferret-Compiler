package parser

import (
	"path/filepath"
	"testing"

	"compiler/ctx"
	"compiler/internal/config"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/testutil"
)

// createTestCompilerContext creates a minimal compiler context for parser testing
// This function is local to parser package to avoid import cycles
func createTestCompilerContext(t *testing.T, entryPointPath string) *ctx.CompilerContext {
	tempDir := testutil.CreateTempProject(t)

	// Create minimal config without depending on config package internals
	projectConfig := &config.ProjectConfig{
		Compiler: config.CompilerConfig{
			Version: "0.1.0-test",
		},
		Cache: config.CacheConfig{
			Path: ".ferret/modules",
		},
		Remote: config.RemoteConfig{
			Enabled: false,
			Share:   false,
		},
		Dependencies: config.DependencyConfig{
			Modules: []string{},
		},
		ProjectRoot: tempDir,
	}

	cachePath := filepath.Join(tempDir, projectConfig.Cache.Path)

	// Get entry point relative to temp dir
	entryPoint, err := filepath.Rel(tempDir, entryPointPath)
	if err != nil {
		// If we can't get relative path, just use the filename
		entryPoint = filepath.Base(entryPointPath)
	}
	entryPoint = filepath.ToSlash(entryPoint)

	return &ctx.CompilerContext{
		EntryPoint:    entryPoint,
		Builtins:      semantic.AddPreludeSymbols(semantic.NewSymbolTable(nil)),
		Modules:       make(map[string]*ctx.Module),
		Reports:       report.Reports{},
		CachePath:     filepath.ToSlash(cachePath),
		ProjectConfig: projectConfig,
		ProjectRoot:   tempDir,
	}
}
