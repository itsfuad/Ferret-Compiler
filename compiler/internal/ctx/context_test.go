package ctx

import (
	"testing"

	"compiler/internal/frontend/ast"
)

func TestModuleFunctions(t *testing.T) {
	// Reset the contextCreated flag for testing
	contextCreated = false
	defer func() { contextCreated = false }()

	ctx := &CompilerContext{}

	// Test GetModule
	if module, _ := ctx.GetModule("test"); module != nil {
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
	if module, _ := ctx.GetModule("test"); module == nil {
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

	// Test FinishParsing
	ctx.FinishParsing("test")
	if ctx.IsModuleParsing("test") {
		t.Error("IsModuleParsing should return false after FinishParsing")
	}
	if len(ctx.ParsingStack) != 0 {
		t.Error("ParsingStack should be empty after FinishParsing")
	}
}

// TestCycleDetection tests the cycle detection functionality
func TestCycleDetection(t *testing.T) {
	ctx := &CompilerContext{}

	// Test that DetectCycle returns false when no cycle exists
	cycle, found := ctx.DetectCycle("A", "B")
	if found {
		t.Error("DetectCycle should return false when no cycle exists")
	}
	if cycle != nil {
		t.Error("DetectCycle should return nil cycle when no cycle exists")
	}

	// Create a dependency chain: A -> B -> C
	ctx.DetectCycle("A", "B") // This should not detect a cycle
	ctx.DetectCycle("B", "C") // This should not detect a cycle

	// Now try to create a cycle: C -> A, which should complete the cycle A -> B -> C -> A
	cycle, found = ctx.DetectCycle("C", "A")
	if !found {
		t.Error("DetectCycle should detect the cycle C -> A")
	}

	// The cycle should start from C (the first module in the cycle path when detected)
	expectedCycle := []string{"C", "A", "B", "C"}
	if len(cycle) != len(expectedCycle) {
		t.Errorf("Expected cycle length %d, got %d", len(expectedCycle), len(cycle))
	}

	for i, expected := range expectedCycle {
		if i >= len(cycle) || cycle[i] != expected {
			t.Errorf("Expected cycle %v, got %v", expectedCycle, cycle)
			break
		}
	}

	// Test direct cycle: A -> B -> A
	ctx2 := &CompilerContext{}
	ctx2.DetectCycle("A", "B")                // Create A -> B
	cycle, found = ctx2.DetectCycle("B", "A") // Try to create B -> A (should detect cycle)

	if !found {
		t.Error("DetectCycle should detect direct cycle B -> A")
	}

	expectedDirectCycle := []string{"B", "A", "B"}
	if len(cycle) != len(expectedDirectCycle) {
		t.Errorf("Expected direct cycle length %d, got %d", len(expectedDirectCycle), len(cycle))
	}

	for i, expected := range expectedDirectCycle {
		if i >= len(cycle) || cycle[i] != expected {
			t.Errorf("Expected direct cycle %v, got %v", expectedDirectCycle, cycle)
			break
		}
	}
}

func TestFullPathToImportPath(t *testing.T) {
	ctx := &CompilerContext{
		ProjectRoot: "/project/root",
	}

	tests := []struct {
		fullPath string
		expected string
	}{
		{"/project/root/src/module.fr", "root/src/module"},
		{"/project/root/main.fr", "root/main"},
		{"/project/root/pkg/sub/file.fr", "root/pkg/sub/file"},
		{"/different/path/file.fr", ""}, // Outside project root
	}

	for _, test := range tests {
		result := ctx.FullPathToImportPath(test.fullPath)
		if result != test.expected {
			t.Errorf("FullPathToModuleName(%s): expected %s, got %s",
				test.fullPath, test.expected, result)
		}
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
		{"/project/root/src/module.fr", "module"},
		{"/project/root/main.fr", "main"},
		{"/project/root/pkg/sub/file.fr", "file"},
		{"/different/path/file.fr", ""}, // Outside project root
	}

	for _, test := range tests {
		result := ctx.FullPathToModuleName(test.fullPath)
		if result != test.expected {
			t.Errorf("FullPathToModuleName(%s): expected %s, got %s",
				test.fullPath, test.expected, result)
		}
	}
}
