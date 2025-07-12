package parser

import (
	"fmt"
	"path/filepath"
	"testing"

	"compiler/ctx"
	"compiler/internal/config"
	"compiler/internal/frontend/ast"
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

func evaluateTestResult(t *testing.T, r interface{}, nodes []ast.Node, desc string, isValid bool) {

	whatsgot := ""
	if r != nil {
		whatsgot += fmt.Sprintf("panic: %s", r)
	}
	if len(nodes) == 0 {
		if whatsgot != "" {
			whatsgot += ", "
		}
		whatsgot += "0 nodes"
	} else {
		//whatsgot = fmt.Sprintf("no panic, %d nodes", len(nodes))
		if whatsgot != "" {
			whatsgot += ", "
		}
		whatsgot += fmt.Sprintf("no panic, %d nodes", len(nodes))
	}

	if isValid && (r != nil || len(nodes) == 0) { // true if panic is nil or nodes are not empty
		t.Errorf("%s: expected no panic or no 0 nodes, got %s", desc, whatsgot)
	} else if !isValid && (r == nil && len(nodes) > 0) { // true if panic is not nil or nodes are empty
		t.Errorf("%s: expected panic or 0 nodes, got %s", desc, whatsgot)
	}
}

func testParseWithPanic(t *testing.T, input string, desc string, isValid bool) {
	t.Helper()
	filePath := testutil.CreateTestFile(t, input)
	ctx := createTestCompilerContext(t, filePath)
	defer ctx.Destroy()

	p := NewParser(filePath, ctx, false)

	nodes := []ast.Node{}

	defer func() {
		evaluateTestResult(t, recover(), nodes, desc, isValid)
	}()

	nodes = p.Parse().Nodes
}
