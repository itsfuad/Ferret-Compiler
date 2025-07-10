package ctx

import (
	"compiler/internal/frontend/ast"
	"reflect"
	"testing"
)

func TestModuleFunctions(t *testing.T) {
	// Reset the contextCreated flag for testing
	contextCreated = false
	defer func() { contextCreated = false }()

	ctx := &CompilerContext{}

	// Test GetModule
	if ctx.GetModule("test") != nil {
		t.Error("GetModule should return nil for non-existent module")
	}

	// Test ModuleCount
	if ctx.ModuleCount() != 0 {
		t.Error("ModuleCount should be 0 for empty context")
	}

	// Test ModuleNames
	if len(ctx.ModuleNames()) != 0 {
		t.Error("ModuleNames should return empty slice for empty context")
	}

	// Test HasModule
	if ctx.HasModule("test") {
		t.Error("HasModule should return false for non-existent module")
	}

	// Test with a mock AST program
	mockAST := &ast.Program{FullPath: "test/path.fr"}
	ctx.AddModule("test", mockAST)

	// Test GetModule after adding
	if ctx.GetModule("test") == nil {
		t.Error("GetModule should return module after adding")
	}

	// Test ModuleCount after adding
	if ctx.ModuleCount() != 1 {
		t.Error("ModuleCount should be 1 after adding a module")
	}

	// Test ModuleNames after adding
	names := ctx.ModuleNames()
	if len(names) != 1 || names[0] != "test" {
		t.Error("ModuleNames should return [test] after adding a module")
	}

	// Test HasModule after adding
	if !ctx.HasModule("test") {
		t.Error("HasModule should return true after adding a module")
	}

	// Test RemoveModule
	ctx.RemoveModule("test")
	if ctx.HasModule("test") {
		t.Error("HasModule should return false after removing a module")
	}
}

func TestParsingFunctions(t *testing.T) {
	ctx := &CompilerContext{}

	// Test IsModuleParsing
	if ctx.IsModuleParsing("test") {
		t.Error("IsModuleParsing should return false for non-existent module")
	}

	// Test StartParsing
	ctx.StartParsing("test")
	if !ctx.IsModuleParsing("test") {
		t.Error("IsModuleParsing should return true after StartParsing")
	}
	if len(ctx.ParsingStack) != 1 || ctx.ParsingStack[0] != "test" {
		t.Error("ParsingStack should contain the module after StartParsing")
	}

	// Test GetCyclePath
	path, found := ctx.GetCyclePath("test")
	if !found {
		t.Error("GetCyclePath should find a cycle for a module being parsed")
	}
	if len(path) != 1 || path[0] != "test" {
		t.Error("GetCyclePath should return the correct path")
	}

	// Test FinishParsing
	ctx.FinishParsing("test")
	if ctx.IsModuleParsing("test") {
		t.Error("IsModuleParsing should return false after FinishParsing")
	}
	if len(ctx.ParsingStack) != 0 {
		t.Error("ParsingStack should be empty after FinishParsing")
	}
}

func TestCycleDetection(t *testing.T) {
	ctx := &CompilerContext{
		DepGraph: make(map[string][]string),
	}

	// Set up a simple dependency graph
	ctx.AddDepEdge("A", "B")
	ctx.AddDepEdge("B", "C")
	ctx.AddDepEdge("C", "D")

	// No cycle
	cycle, found := ctx.DetectCycle("A")
	if found {
		t.Errorf("Shouldn't detect cycle in acyclic graph, got: %v", cycle)
	}

	// Create a cycle
	ctx.AddDepEdge("D", "B")

	// Should detect cycle
	cycle, found = ctx.DetectCycle("A")
	if !found {
		t.Error("Failed to detect cycle in cyclic graph")
	}

	// Verify the cycle contains the right nodes
	expectedCycle := []string{"B", "C", "D", "B"}
	if !reflect.DeepEqual(cycle, expectedCycle) {
		t.Errorf("Expected cycle %v, got %v", expectedCycle, cycle)
	}
}

func TestFullPathToModuleName(t *testing.T) {
	ctx := &CompilerContext{
		ProjectRoot: "/project/root",
	}

	tests := []struct {
		fullPath string
		expected string
	}{
		{"/project/root/src/module.fr", "src/module"},
		{"/project/root/main.fr", "main"},
		{"/project/root/pkg/sub/file.fr", "pkg/sub/file"},
		{"/different/path/file.fr", "../../different/path/file"}, // Outside project root
	}

	for _, test := range tests {
		result := ctx.FullPathToModuleName(test.fullPath)
		if result != test.expected {
			t.Errorf("FullPathToModuleName(%s): expected %s, got %s",
				test.fullPath, test.expected, result)
		}
	}
}
