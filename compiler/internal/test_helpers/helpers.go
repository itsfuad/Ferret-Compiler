package test_helpers

import (
	"compiler/ctx"
	"compiler/internal/config"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"os"
	"path/filepath"
	"testing"
)

// CreateTestFile creates a temporary test file and returns its path
func CreateTestFile(t *testing.T) string {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.fer")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()
	return filePath
}

func CreateTestFileWithContent(t *testing.T, content string) string {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.fer")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// CreateTestCompilerContext creates a minimal compiler context for testing
func CreateTestCompilerContext(t *testing.T, entryPointPath string) *ctx.CompilerContext {
	// Create a temporary directory that will act as the project root
	tempDir := t.TempDir()

	// Create a default project config for testing
	projectConfig := &config.ProjectConfig{
		Compiler: config.CompilerConfig{
			Version: "0.1.0",
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

	// Create cache directory
	cachePath := filepath.Join(tempDir, projectConfig.Cache.Path)
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

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
