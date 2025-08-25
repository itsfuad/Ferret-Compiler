package ctx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"compiler/config"
	"compiler/internal/frontend/ast"
	"compiler/internal/modules"
	"compiler/internal/utils/stack"
)

// Test constants to avoid duplicated literals
const (
	testModulePath     = "test/module"
	testFilePath       = "/test/path.fer"
	testProjectName    = "test"
	testProjectVersion = "1.0.0"
	mainEntry          = "main.fer"
	mainOutput         = "main.exe"
	ferretCacheDir     = ".ferret"
	expectedPanicMsg   = "Expected panic but didn't get one"
	expectedNonNilCtx  = "Expected non-nil context"
)

// Test helper to create a temporary project config
func createTestProjectConfig(t *testing.T, name, root string) *config.ProjectConfig {
	t.Helper()
	return &config.ProjectConfig{
		Name: name,
		Compiler: config.CompilerConfig{
			Version: testProjectVersion,
		},
		Build: config.BuildConfig{
			Entry:  mainEntry,
			Output: mainOutput,
		},
		Cache: config.CacheConfig{
			Path: ferretCacheDir,
		},
		External: config.ExternalConfig{
			AllowSharing:        true,
			AllowRemoteImport:   true,
			AllowExternalImport: true,
		},
		Dependencies: config.DependencyConfig{
			Packages: make(map[string]string),
		},
		Neighbors: config.NeighborConfig{
			Projects: make(map[string]string),
		},
		ProjectRoot: root,
	}
}

// Test helper to create a temporary directory
func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ferret-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// Test helper to reset the context creation flag
func resetContextFlag() {
	contextCreated = false
}

// Test helper to create a minimal compiler context for testing
func createTestCompilerContext(t *testing.T, projectConfig *config.ProjectConfig) *CompilerContext {
	t.Helper()

	if contextCreated {
		panic("Cannot create context: context already exists")
	}

	resetContextFlag()

	// Mark context as created to prevent double creation
	contextCreated = true

	entryPoint := filepath.Join(projectConfig.ProjectRoot, projectConfig.Build.Entry)
	entryPoint = filepath.ToSlash(entryPoint)

	remoteCachePath := filepath.Join(projectConfig.ProjectRoot, projectConfig.Cache.Path)
	remoteCachePath = filepath.ToSlash(remoteCachePath)
	os.MkdirAll(remoteCachePath, 0755)

	return &CompilerContext{
		EntryPoint:          entryPoint,
		Builtins:            nil, // Simplified for tests
		Modules:             make(map[string]*modules.Module),
		Reports:             nil,
		ProjectConfig:       projectConfig,
		ProjectStack:        stack.New[*config.ProjectConfig](), // Initialize proper stack
		RemoteCachePath:     remoteCachePath,
		BuiltinModules:      make(map[string]string), // Empty for tests
		ProjectRootFullPath: projectConfig.ProjectRoot,
	}
}

// Helper function to create test cases for AddModule
func createAddModuleTestCases() []struct {
	name        string
	importPath  string
	program     *ast.Program
	expectPanic bool
} {
	// Create a test AST program
	program := &ast.Program{
		FullPath:   testFilePath,
		ImportPath: testModulePath,
		Alias:      testProjectName,
		Nodes:      []ast.Node{},
	}

	return []struct {
		name        string
		importPath  string
		program     *ast.Program
		expectPanic bool
	}{
		{
			name:        "valid module addition",
			importPath:  testModulePath,
			program:     program,
			expectPanic: false,
		},
		{
			name:        "nil program should panic",
			importPath:  "test/nil-module",
			program:     nil,
			expectPanic: true,
		},
		{
			name:        "duplicate module addition should not panic",
			importPath:  testModulePath, // Same as first test
			program:     program,
			expectPanic: false,
		},
	}
}

// Helper function to test module addition with panic handling
func testModuleAddition(t *testing.T, ctx *CompilerContext, importPath string, program *ast.Program, expectPanic bool) {
	if expectPanic {
		defer func() {
			if r := recover(); r == nil {
				t.Error(expectedPanicMsg)
			}
		}()
	}

	ctx.AddModule(importPath, program)

	if !expectPanic {
		validateAddedModule(t, ctx, importPath)
	}
}

// Helper function to validate added module
func validateAddedModule(t *testing.T, ctx *CompilerContext, importPath string) {
	module, err := ctx.GetModule(importPath)
	if err != nil {
		t.Errorf("Failed to get added module: %v", err)
	}
	if module.Phase != modules.PHASE_PARSED {
		t.Errorf("Expected module phase to be PHASE_PARSED, got %v", module.Phase)
	}
}

func TestCompilerContextAddModule(t *testing.T) {
	projectConfig := createTestProjectConfig(t, testProjectName, createTempDir(t))
	ctx := createTestCompilerContext(t, projectConfig)
	defer ctx.Destroy()

	tests := createAddModuleTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testModuleAddition(t, ctx, tt.importPath, tt.program, tt.expectPanic)
		})
	}
}

// Helper function to test empty context operations
func testEmptyContextOperations(t *testing.T, ctx *CompilerContext) {
	if ctx.ModuleCount() != 0 {
		t.Errorf("Expected 0 modules, got %d", ctx.ModuleCount())
	}

	if ctx.HasModule("nonexistent") {
		t.Error("Expected false for nonexistent module")
	}

	if ctx.IsModuleParsed("nonexistent") {
		t.Error("Expected false for nonexistent parsed module")
	}

	phase := ctx.GetModulePhase("nonexistent")
	if phase != modules.PHASE_NOT_STARTED {
		t.Errorf("Expected PHASE_NOT_STARTED, got %v", phase)
	}
}

// Helper function to test populated context operations
func testPopulatedContextOperations(t *testing.T, ctx *CompilerContext, testModules []string) {
	if ctx.ModuleCount() != len(testModules) {
		t.Errorf("Expected %d modules, got %d", len(testModules), ctx.ModuleCount())
	}

	for _, modName := range testModules {
		if !ctx.HasModule(modName) {
			t.Errorf("Expected module %s to exist", modName)
		}

		if !ctx.IsModuleParsed(modName) {
			t.Errorf("Expected module %s to be parsed", modName)
		}

		phase := ctx.GetModulePhase(modName)
		if phase != modules.PHASE_PARSED {
			t.Errorf("Expected PHASE_PARSED for %s, got %v", modName, phase)
		}
	}

	// Test module names
	names := ctx.ModuleNames()
	if len(names) != len(testModules) {
		t.Errorf("Expected %d module names, got %d", len(testModules), len(names))
	}
}

func TestCompilerContextModuleOperations(t *testing.T) {
	projectConfig := createTestProjectConfig(t, testProjectName, createTempDir(t))
	ctx := createTestCompilerContext(t, projectConfig)
	defer ctx.Destroy()

	program := &ast.Program{
		FullPath:   testFilePath,
		ImportPath: testModulePath,
		Alias:      testProjectName,
		Nodes:      []ast.Node{},
	}

	// Test with empty context
	t.Run("empty context operations", func(t *testing.T) {
		testEmptyContextOperations(t, ctx)
	})

	// Add some modules
	testModules := []string{"module1", "module2", "module3"}
	for _, modName := range testModules {
		ctx.AddModule(modName, program)
	}

	t.Run("populated context operations", func(t *testing.T) {
		testPopulatedContextOperations(t, ctx, testModules)
	})
}

// Helper function to create module phase test cases
func createModulePhaseTestCases() []struct {
	name          string
	setPhase      modules.ModulePhase
	canProcess    []modules.ModulePhase
	cannotProcess []modules.ModulePhase
} {
	return []struct {
		name          string
		setPhase      modules.ModulePhase
		canProcess    []modules.ModulePhase
		cannotProcess []modules.ModulePhase
	}{
		{
			name:          "parsed phase",
			setPhase:      modules.PHASE_PARSED,
			canProcess:    []modules.ModulePhase{modules.PHASE_COLLECTED},
			cannotProcess: []modules.ModulePhase{modules.PHASE_RESOLVED, modules.PHASE_TYPECHECKED},
		},
		{
			name:          "collected phase",
			setPhase:      modules.PHASE_COLLECTED,
			canProcess:    []modules.ModulePhase{modules.PHASE_RESOLVED},
			cannotProcess: []modules.ModulePhase{modules.PHASE_PARSED, modules.PHASE_TYPECHECKED},
		},
		{
			name:          "resolved phase",
			setPhase:      modules.PHASE_RESOLVED,
			canProcess:    []modules.ModulePhase{modules.PHASE_TYPECHECKED},
			cannotProcess: []modules.ModulePhase{modules.PHASE_PARSED, modules.PHASE_COLLECTED},
		},
	}
}

// Helper function to test phase transitions
func testPhaseTransitions(t *testing.T, ctx *CompilerContext, moduleName string, tt struct {
	name          string
	setPhase      modules.ModulePhase
	canProcess    []modules.ModulePhase
	cannotProcess []modules.ModulePhase
}) {
	ctx.SetModulePhase(moduleName, tt.setPhase)

	currentPhase := ctx.GetModulePhase(moduleName)
	if currentPhase != tt.setPhase {
		t.Errorf("Expected phase %v, got %v", tt.setPhase, currentPhase)
	}

	for _, phase := range tt.canProcess {
		if !ctx.CanProcessPhase(moduleName, phase) {
			t.Errorf("Expected to be able to process phase %v from %v", phase, tt.setPhase)
		}
	}

	for _, phase := range tt.cannotProcess {
		if ctx.CanProcessPhase(moduleName, phase) {
			t.Errorf("Expected NOT to be able to process phase %v from %v", phase, tt.setPhase)
		}
	}
}

func TestCompilerContextModulePhases(t *testing.T) {
	projectConfig := createTestProjectConfig(t, testProjectName, createTempDir(t))
	ctx := createTestCompilerContext(t, projectConfig)
	defer ctx.Destroy()

	program := &ast.Program{
		FullPath:   testFilePath,
		ImportPath: testModulePath,
		Alias:      testProjectName,
		Nodes:      []ast.Node{},
	}

	moduleName := testModulePath
	ctx.AddModule(moduleName, program)

	tests := createModulePhaseTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPhaseTransitions(t, ctx, moduleName, tt)
		})
	}
}

func TestCompilerContextIsRemoteImport(t *testing.T) {
	projectConfig := createTestProjectConfig(t, "test", createTempDir(t))
	ctx := createTestCompilerContext(t, projectConfig)
	defer ctx.Destroy()

	tests := []struct {
		importPath string
		expected   bool
	}{
		{"github.com/user/repo", true},
		{"gitlab.com/user/repo", true},
		{"bitbucket.org/user/repo", true},
		{"local/module", false},
		{"./relative/path", false},
		{"std/io", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.importPath, func(t *testing.T) {
			result := ctx.IsRemoteImport(tt.importPath)
			if result != tt.expected {
				t.Errorf("IsRemoteImport(%q) = %v, want %v", tt.importPath, result, tt.expected)
			}
		})
	}
}

func TestCompilerContextCachePathToImportPath(t *testing.T) {
	tempDir := createTempDir(t)
	projectConfig := createTestProjectConfig(t, "test", tempDir)
	ctx := createTestCompilerContext(t, projectConfig)
	defer ctx.Destroy()

	tests := []struct {
		name      string
		cachePath string
		expected  string
	}{
		{
			name:      "github module",
			cachePath: filepath.Join(ctx.RemoteCachePath, "github.com", "user", "repo@v1.0.0", "module.fer"),
			expected:  "github.com/user/repo/module",
		},
		{
			name:      "nested module",
			cachePath: filepath.Join(ctx.RemoteCachePath, "github.com", "user", "repo@v1.0.0", "sub", "module.fer"),
			expected:  "github.com/user/repo/sub/module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.CachePathToImportPath(tt.cachePath)
			if result != tt.expected {
				t.Errorf("CachePathToImportPath(%q) = %q, want %q", tt.cachePath, result, tt.expected)
			}
		})
	}
}

func TestCompilerContextFullPathToAlias(t *testing.T) {
	projectConfig := createTestProjectConfig(t, "test", createTempDir(t))
	ctx := createTestCompilerContext(t, projectConfig)
	defer ctx.Destroy()

	tests := []struct {
		fullPath string
		expected string
	}{
		{"/path/to/module.fer", "module"},
		{"/path/to/nested/file.fer", "file"},
		{"C:\\Windows\\path\\test.fer", "test"},
		{"simple.fer", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.fullPath, func(t *testing.T) {
			result := ctx.FullPathToAlias(tt.fullPath)
			if result != tt.expected {
				t.Errorf("FullPathToAlias(%q) = %q, want %q", tt.fullPath, result, tt.expected)
			}
		})
	}
}

// Helper function to create cycle test cases
func createCycleTestCases() []struct {
	name         string
	edges        [][]string
	expectCycle  bool
	expectedPath []string
} {
	return []struct {
		name         string
		edges        [][]string
		expectCycle  bool
		expectedPath []string
	}{
		{
			name:        "no cycle - linear dependency",
			edges:       [][]string{{"A", "B"}, {"B", "C"}, {"C", "D"}},
			expectCycle: false,
		},
		{
			name:         "simple cycle",
			edges:        [][]string{{"A", "B"}, {"B", "A"}},
			expectCycle:  true,
			expectedPath: []string{"B", "A", "B"},
		},
		{
			name:         "complex cycle",
			edges:        [][]string{{"A", "B"}, {"B", "C"}, {"C", "A"}},
			expectCycle:  true,
			expectedPath: []string{"C", "A", "B", "C"},
		},
		{
			name:        "no cycle - tree structure",
			edges:       [][]string{{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "E"}},
			expectCycle: false,
		},
	}
}

// Helper function to process edges and detect cycles
func processEdgesForCycle(ctx *CompilerContext, edges [][]string) ([]string, bool) {
	var cyclePath []string
	var hasCycle bool

	for i, edge := range edges {
		cycle, detected := ctx.DetectCycle(edge[0], edge[1])
		if detected {
			hasCycle = true
			cyclePath = cycle
			break
		}
		// If no cycle, verify edge was added (for non-final edges)
		if !detected && i < len(edges)-1 {
			if !isEdgeAdded(ctx, edge[0], edge[1]) {
				// Edge should have been added if no cycle detected
			}
		}
	}
	return cyclePath, hasCycle
}

// Helper function to check if edge was added to dependency graph
func isEdgeAdded(ctx *CompilerContext, from, to string) bool {
	deps, exists := ctx.DepGraph[from]
	if !exists {
		return false
	}
	for _, dep := range deps {
		if dep == to {
			return true
		}
	}
	return false
}

// Helper function to validate cycle path
func validateCyclePath(t *testing.T, cyclePath, expectedPath []string, expectCycle bool) {
	if !expectCycle || len(expectedPath) == 0 {
		return
	}

	if len(cyclePath) != len(expectedPath) {
		t.Errorf("Expected cycle path length %d, got %d. Actual path: %v", len(expectedPath), len(cyclePath), cyclePath)
		return
	}

	for i, expected := range expectedPath {
		if i < len(cyclePath) && cyclePath[i] != expected {
			t.Errorf("Expected cycle path[%d] = %s, got %s. Full path: %v", i, expected, cyclePath[i], cyclePath)
		}
	}
}

func TestCompilerContextDetectCycle(t *testing.T) {
	projectConfig := createTestProjectConfig(t, testProjectName, createTempDir(t))
	ctx := createTestCompilerContext(t, projectConfig)
	defer ctx.Destroy()

	tests := createCycleTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset context for each test
			ctx.DepGraph = make(map[string][]string)

			cyclePath, hasCycle := processEdgesForCycle(ctx, tt.edges)

			if hasCycle != tt.expectCycle {
				t.Errorf("Expected cycle detection to be %v, got %v", tt.expectCycle, hasCycle)
			}

			validateCyclePath(t, cyclePath, tt.expectedPath, tt.expectCycle)
		})
	}
}

func TestCompilerContextContextLifecycle(t *testing.T) {
	// Test context creation and destruction
	t.Run("single context lifecycle", func(t *testing.T) {
		projectConfig := createTestProjectConfig(t, "test", createTempDir(t))
		ctx := createTestCompilerContext(t, projectConfig)

		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}

		// Context should be marked as created
		if !contextCreated {
			t.Error("Expected contextCreated to be true")
		}

		ctx.Destroy()

		// Context should be marked as destroyed
		if contextCreated {
			t.Error("Expected contextCreated to be false after destroy")
		}
	})

	t.Run("multiple context creation should panic", func(t *testing.T) {
		projectConfig := createTestProjectConfig(t, "test", createTempDir(t))
		ctx1 := createTestCompilerContext(t, projectConfig)
		defer ctx1.Destroy()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when creating second context")
			}
		}()

		// This should panic because contextCreated is already true
		createTestCompilerContext(t, projectConfig)
	})

	t.Run("destroy non-created context", func(t *testing.T) {
		resetContextFlag()

		ctx := &CompilerContext{}
		// Should not panic
		ctx.Destroy()
	})
}

// Helper function to create import path test cases
func createImportPathTestCases() []struct {
	name          string
	importPath    string
	expectedType  modules.ModuleType
	expectError   bool
	errorContains string
} {
	return []struct {
		name          string
		importPath    string
		expectedType  modules.ModuleType
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty import path",
			importPath:    "",
			expectedType:  modules.UNKNOWN,
			expectError:   true,
			errorContains: "empty import path",
		},
		{
			name:         "local module",
			importPath:   "testproject/submodule",
			expectedType: modules.LOCAL,
			expectError:  false,
		},
		{
			name:          "remote module not in dependencies",
			importPath:    "github.com/unknown/repo/module",
			expectedType:  modules.REMOTE,
			expectError:   true,
			errorContains: "is not installed",
		},
	}
}

// Helper function to setup project config for import path testing
func setupProjectConfigForImportPath(t *testing.T) (*config.ProjectConfig, *CompilerContext) {
	tempDir := createTempDir(t)
	projectConfig := createTestProjectConfig(t, "testproject", tempDir)

	// Add some neighbors
	projectConfig.Neighbors.Projects = map[string]string{
		"neighbor1": "../neighbor1",
	}

	// Add some dependencies
	projectConfig.Dependencies.Packages = map[string]string{
		"github.com/user/repo": "v1.0.0",
	}

	ctx := createTestCompilerContext(t, projectConfig)
	// Push current project to stack
	ctx.ProjectStack.Push(projectConfig)

	return projectConfig, ctx
}

// Helper function to validate import path resolution result
func validateImportPathResult(t *testing.T, tt struct {
	name          string
	importPath    string
	expectedType  modules.ModuleType
	expectError   bool
	errorContains string
}, projectConfig *config.ProjectConfig, resolvedPath string, modType modules.ModuleType, err error) {

	if tt.expectError {
		if err == nil {
			t.Errorf("Expected error but got none")
		} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
			t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
		}
	} else if err != nil && !strings.Contains(err.Error(), "does not exist") {
		// Allow "does not exist" errors since we're not creating actual files
		t.Errorf("Unexpected error: %v", err)
	}

	if modType != tt.expectedType {
		t.Errorf("Expected module type %v, got %v", tt.expectedType, modType)
	}

	if !tt.expectError && err == nil {
		if projectConfig == nil {
			t.Error("Expected non-nil project config")
		}
		if resolvedPath == "" {
			t.Error("Expected non-empty resolved path")
		}
	}
}

func TestCompilerContextResolveImportPath(t *testing.T) {
	_, ctx := setupProjectConfigForImportPath(t)
	defer ctx.Destroy()

	tests := createImportPathTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectConfig, resolvedPath, modType, err := ctx.ResolveImportPath(tt.importPath)
			validateImportPathResult(t, tt, projectConfig, resolvedPath, modType, err)
		})
	}
}
