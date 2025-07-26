package ctx

import (
	"os"
	"path/filepath"
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
	ctx.AddModule("test", mockAST, false) // Test module is not builtin

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
	if ctx.isModuleParsing("test") {
		t.Error("IsModuleParsing should return false for non-existent module")
	}

	// Test StartParsing
	ctx.StartParsing("test")
	if !ctx.isModuleParsing("test") {
		t.Error("IsModuleParsing should return true after StartParsing")
	}
	if len(ctx._parsingStack) != 1 || ctx._parsingStack[0] != "test" {
		t.Error("ParsingStack should contain the module after StartParsing")
	}

	// Test FinishParsing
	ctx.FinishParsing("test")
	if ctx.isModuleParsing("test") {
		t.Error("IsModuleParsing should return false after FinishParsing")
	}
	if len(ctx._parsingStack) != 0 {
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

func TestIsRemoteModuleCachedFlat(t *testing.T) {
	tests := []struct {
		name           string
		flatModuleName string
		createModule   bool
		expected       bool
	}{
		{
			name:           "existing flat module",
			flatModuleName: "github.com/user/repo@v1.0.0",
			createModule:   true,
			expected:       true,
		},
		{
			name:           "non-existing flat module",
			flatModuleName: "github.com/user/nonexistent@v1.0.0",
			createModule:   false,
			expected:       false,
		},
		{
			name:           "empty module name",
			flatModuleName: "",
			createModule:   false,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "ferret-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Create context with temporary cache path
			context := &CompilerContext{
				RemoteCachePath: tempDir,
			}

			// Create module directory if specified
			if tt.createModule && tt.flatModuleName != "" {
				moduleDir := filepath.Join(tempDir, tt.flatModuleName)
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					t.Fatal(err)
				}
			}

			// Test IsRemoteModuleCachedFlat
			result := context.IsRemoteModuleCachedFlat(tt.flatModuleName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetRemoteModuleCachePathFlat(t *testing.T) {
	tests := []struct {
		name           string
		flatModuleName string
		basePath       string
		expected       string
	}{
		{
			name:           "standard flat module path",
			flatModuleName: "github.com/user/repo@v1.0.0",
			basePath:       "/cache/modules",
			expected:       "/cache/modules/github.com/user/repo@v1.0.0",
		},
		{
			name:           "complex version flat module",
			flatModuleName: "github.com/org/project@v2.1.0-beta.1",
			basePath:       "/tmp/cache",
			expected:       "/tmp/cache/github.com/org/project@v2.1.0-beta.1",
		},
		{
			name:           "empty module name",
			flatModuleName: "",
			basePath:       "/cache",
			expected:       "/cache/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := &CompilerContext{
				RemoteCachePath: tt.basePath,
			}

			result := context.GetRemoteModuleCachePathFlat(tt.flatModuleName)

			// Use filepath.Join for platform-independent comparison
			expected := filepath.Join(tt.basePath, tt.flatModuleName)
			if result != expected {
				t.Errorf("Expected %s, got %s", expected, result)
			}
		})
	}
}

func TestParseRemoteImport(t *testing.T) {
	tests := []struct {
		name         string
		importPath   string
		expectedRepo string
		expectedVer  string
		expectedSub  string
	}{
		{
			name:         "basic GitHub import",
			importPath:   "github.com/user/repo",
			expectedRepo: "github.com/user/repo",
			expectedVer:  "latest",
			expectedSub:  "",
		},
		{
			name:         "GitHub import with version",
			importPath:   "github.com/user/repo@v1.2.3",
			expectedRepo: "github.com/user/repo",
			expectedVer:  "v1.2.3",
			expectedSub:  "",
		},
		{
			name:         "GitHub import with subpath",
			importPath:   "github.com/user/repo/utils/math",
			expectedRepo: "github.com/user/repo",
			expectedVer:  "latest",
			expectedSub:  "utils/math",
		},
		{
			name:         "GitHub import with version and subpath",
			importPath:   "github.com/user/repo@v2.0.0/data/types",
			expectedRepo: "github.com/user/repo",
			expectedVer:  "v2.0.0",
			expectedSub:  "data/types",
		},
		{
			name:         "invalid import path",
			importPath:   "invalid/path",
			expectedRepo: "",
			expectedVer:  "",
			expectedSub:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := &CompilerContext{}

			repo, version, subpath := context.ParseRemoteImport(tt.importPath)

			if repo != tt.expectedRepo {
				t.Errorf("Expected repo %s, got %s", tt.expectedRepo, repo)
			}
			if version != tt.expectedVer {
				t.Errorf("Expected version %s, got %s", tt.expectedVer, version)
			}
			if subpath != tt.expectedSub {
				t.Errorf("Expected subpath %s, got %s", tt.expectedSub, subpath)
			}
		})
	}
}

func TestFlatVsOldCacheStructure(t *testing.T) {
	// Test that demonstrates the difference between old nested and new flat structure
	tempDir, err := os.MkdirTemp("", "ferret-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	context := &CompilerContext{
		RemoteCachePath: tempDir,
	}

	// Test flat structure paths
	flatModules := []string{
		"github.com/user/repo@v1.0.0",
		"github.com/user/repo@v2.0.0",
		"github.com/other/lib@v1.5.0",
	}

	for _, flatName := range flatModules {
		// Create flat module directories
		moduleDir := context.GetRemoteModuleCachePathFlat(flatName)
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Verify they exist and are detected
		if !context.IsRemoteModuleCachedFlat(flatName) {
			t.Errorf("Flat module %s should be detected as cached", flatName)
		}
	}

	// Verify different versions of same repo can coexist
	repo1v1 := "github.com/user/repo@v1.0.0"
	repo1v2 := "github.com/user/repo@v2.0.0"

	if !context.IsRemoteModuleCachedFlat(repo1v1) {
		t.Error("Version v1.0.0 should be cached")
	}
	if !context.IsRemoteModuleCachedFlat(repo1v2) {
		t.Error("Version v2.0.0 should be cached")
	}

	// Verify paths are different
	path1 := context.GetRemoteModuleCachePathFlat(repo1v1)
	path2 := context.GetRemoteModuleCachePathFlat(repo1v2)

	if path1 == path2 {
		t.Error("Different versions should have different cache paths")
	}
}
