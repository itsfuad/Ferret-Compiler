package parser

import (
	"fmt"
	"path/filepath"

	//"runtime/debug"
	"testing"

	"compiler/config"
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/symbol"
	"compiler/internal/testutil"
	"compiler/internal/utils/stack"
	"compiler/report"
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
			Path: ".ferret",
		},
		Dependencies: config.DependencyConfig{
			Packages: make(map[string]string),
		},
		ProjectRoot: tempDir,
	}

	// Get entry point relative to temp dir
	entryPoint, err := filepath.Rel(tempDir, entryPointPath)
	if err != nil {
		// If we can't get relative path, just use the filename
		entryPoint = filepath.Base(entryPointPath)
	}
	entryPoint = filepath.ToSlash(entryPoint)

	return &ctx.CompilerContext{
		EntryPoint:          entryPoint,
		Builtins:            symbol.AddPreludeSymbols(symbol.NewSymbolTable(nil)),
		Modules:             make(map[string]*modules.Module),
		Reports:             report.Reports{},
		ProjectConfig:       projectConfig,
		ProjectStack:        stack.New[*config.ProjectConfig](),
		ProjectRootFullPath: tempDir,
	}
}

func evaluateTestResult(t *testing.T, nodes []ast.Node, ctx *ctx.CompilerContext, desc string, isValid bool) {

	whatsgot := fmt.Sprintf("%d nodes", len(nodes))

	shouldstop := ctx.Reports.ShouldStopCompilation()

	if shouldstop {
		if whatsgot != "" {
			whatsgot += ", "
		}
		whatsgot += "should stop compilation"
	}

	if isValid {
		if len(nodes) == 0 || shouldstop {
			t.Errorf("%s: expected some nodes, and no stop flag, got %s", desc, whatsgot)
		}
	} else {
		if len(nodes) > 0 && !shouldstop {
			t.Errorf("%s: expected 0 nodes or stop flag, got %s, Nodes: %v, first: %#v", desc, whatsgot, nodes, nodes[0])
		}
	}
}

func testParseWithPanic(t *testing.T, input string, desc string, isValid bool) {
	t.Helper()
	filePath := testutil.CreateTestFile(t, input)
	ctx := createTestCompilerContext(t, filePath)
	defer ctx.Destroy()

	p := NewParser(filePath, ctx, false)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()

	nodes := p.Parse().Nodes
	evaluateTestResult(t, nodes, ctx, desc, isValid)
}
