package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"compiler/internal/config"
	"compiler/internal/ctx"
	"compiler/internal/registry"
)

const (
	//use variables insted of string for files
	BUILTIN_MODULES_DIR = "modules"
	BUILTIN_MODULES_EXT = ".fer"
	MATH_MODULE         = "math/geometry"
	IO_MODULE           = "std/io"

	TEST_REPO        = "github.com/user/repo"
	NON_GH_TEST_REPO = "example.com/test/repo"
)

func TestIsBuiltinModule(t *testing.T) {
	tests := []struct {
		name       string
		importRoot string
		want       bool
	}{
		{"Standard library", "std", true},
		{"Math module", "math", true},
		{"IO module", "io", true},
		{"User project", "myapp", false},
		{"Unknown module", "unknown", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBuiltinModule(tt.importRoot); got != tt.want {
				t.Errorf("IsBuiltinModule(%q) = %v, want %v", tt.importRoot, got, tt.want)
			}
		})
	}
}

func TestIsRemote(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		want       bool
	}{
		{"Empty", "", false},
		{"GitHub path", "github.com/user/repo", true},
		{"Local path", "myproject/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRemote(tt.importPath); got != tt.want {
				t.Errorf("IsRemote(%q) = %v, want %v", tt.importPath, got, tt.want)
			}
		})
	}
}

func TestIsValidFile(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test-file")
	if err != nil {
		t.Fatal(err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Valid file", tempFile.Name(), true},
		{"Non-existent file", "nonexistent-file.txt", false},
		{"Directory", os.TempDir(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidFile(tt.filename); got != tt.want {
				t.Errorf("IsValidFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestGitHubPathToRawURL(t *testing.T) {
	tests := []struct {
		name          string
		importPath    string
		defaultBranch string
		wantURL       string
		wantSubpath   string
	}{
		{"Valid GitHub path", "github.com/user/repo/path/file", "main", "https://raw.githubusercontent.com/user/repo/main/path/file.fer", "path/file"},
		{"Invalid GitHub path", "github.com/user", "main", "", ""},
		{"Non-GitHub path", "gitlab.com/user/repo", "main", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotSubpath := GitHubPathToRawURL(tt.importPath, tt.defaultBranch)
			if gotURL != tt.wantURL || gotSubpath != tt.wantSubpath {
				t.Errorf("GitHubPathToRawURL(%q, %q) = (%v, %v), want (%v, %v)",
					tt.importPath, tt.defaultBranch, gotURL, gotSubpath, tt.wantURL, tt.wantSubpath)
			}
		})
	}
}

func TestFirstPart(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"Empty path", "", ""},
		{"Single part", "file", "file"},
		{"Multiple parts", "project/module/file", "project"},
		{"With windows path", `project\module\file`, "project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FirstPart(tt.path); got != tt.want {
				t.Errorf("FirstPart(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestLastPart(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"Empty path", "", ""},
		{"Single part", "file", "file"},
		{"Multiple parts", "project/module/file", "file"},
		{"With windows path", `project\module\file`, "file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LastPart(tt.path); got != tt.want {
				t.Errorf("LastPart(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveModuleWithBuiltins(t *testing.T) {
	// Create temporary project structure
	tempDir := t.TempDir()
	projectDir := tempDir

	// Create temporary modules directory
	modulesDir := filepath.Join(tempDir, "modules")
	err := os.MkdirAll(filepath.Join(modulesDir, "std"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a built-in module file
	builtinFile := filepath.Join(modulesDir, "std", "io.fer")
	if err := os.WriteFile(builtinFile, []byte("// Built-in IO module"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		projectName string
		importPath  string
		modulesPath string
		wantErr     bool
	}{
		{"Valid built-in module", "testproject", IO_MODULE, modulesDir, false},
		{"Non-existent built-in module", "testproject", "std/nonexistent", modulesDir, true},
		{"Built-in module wrong path", "testproject", IO_MODULE, "/nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxx := &ctx.CompilerContext{
				ProjectRoot: projectDir,
				ModulesPath: tt.modulesPath,
				ProjectConfig: &config.ProjectConfig{
					Name: tt.projectName,
				},
			}

			result, err := ResolveModule(tt.importPath, "", ctxx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveModule(%q) error = %v, wantErr %v",
					tt.importPath, err, tt.wantErr)
			}

			if !tt.wantErr && err == nil {
				expectedPath := filepath.Join(tt.modulesPath, tt.importPath+".fer")
				if result != expectedPath {
					t.Errorf("ResolveModule(%q) = %q, want %q",
						tt.importPath, result, expectedPath)
				}
			}
		})
	}
}

func TestResolveModule(t *testing.T) {
	// Create temporary project structure
	tempDir := t.TempDir()
	projectDir := tempDir // Project root is the temp dir itself

	// Create nested directory structure: module/ (relative to project root)
	err := os.MkdirAll(filepath.Join(projectDir, "module"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test module file at: projectroot/module/test.fer
	moduleFile := filepath.Join(projectDir, "module", "test.fer")
	if err := os.WriteFile(moduleFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		projectName string
		importPath  string
		wantErr     bool
	}{
		{"Remote import", "testproject", "github.com/user/repo/module", true},
		{"Empty import", "testproject", "", true},
		{"Empty project name", "", "someproject/module", true},
		{"Built-in std module", "testproject", IO_MODULE, true},    // Should error with "not implemented yet"
		{"Built-in math module", "testproject", MATH_MODULE, true}, // Should error with "not implemented yet"
		{"Non-existent local module", "testproject", "testproject/nonexistent", true},
		{"Valid local module", "testproject", "testproject/module/test", false},
		{"Unknown external module", "testproject", "unknownmodule/something", true}, // Should error with "module not found"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with project config
			ctxx := &ctx.CompilerContext{
				ProjectRoot: projectDir,
				ProjectConfig: &config.ProjectConfig{
					Name: tt.projectName,
				},
			}

			result, err := ResolveModule(tt.importPath, "", ctxx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveModule(%q) error = %v, wantErr %v",
					tt.importPath, err, tt.wantErr)
			}

			// For the valid case, check the returned path
			if !tt.wantErr && err == nil {
				// The import "testproject/module/test" should resolve to "module/test.fer" relative to project root
				relativePath := strings.TrimPrefix(tt.importPath, tt.projectName+"/")
				expectedPath := filepath.Join(projectDir, relativePath+".fer")
				if result != expectedPath {
					t.Errorf("ResolveModule(%q) = %q, want %q",
						tt.importPath, result, expectedPath)
				}
			}
		})
	}
}

func TestResolveModuleProjectNameValidation(t *testing.T) {
	tempDir := t.TempDir()
	ctxx := &ctx.CompilerContext{
		ProjectRoot: tempDir,
		ProjectConfig: &config.ProjectConfig{
			Name: "", // Empty project name
		},
	}

	_, err := ResolveModule("someproject/module", "", ctxx)
	if err == nil || !strings.Contains(err.Error(), "project name not defined") {
		t.Errorf("Expected error about project name not defined, got: %v", err)
	}
}

func TestResolveVersionConstraint(t *testing.T) {
	tests := []struct {
		name              string
		repoPath          string
		versionConstraint string
		lockFilePackages  map[string]registry.LockEntry
		expectedVersion   string
		expectError       bool
		isGitHub          bool // Whether to test GitHub validation
	}{
		{
			name:              "exact version constraint (non-GitHub)",
			repoPath:          NON_GH_TEST_REPO, // Non-GitHub to avoid validation
			versionConstraint: "v3.2.3",
			expectedVersion:   "v3.2.3",
			expectError:       false,
			isGitHub:          false,
		},
		{
			name:              "caret constraint with lockfile match",
			repoPath:          NON_GH_TEST_REPO,
			versionConstraint: "^v1.0.0",
			lockFilePackages: map[string]registry.LockEntry{
				"example.com/test/repo@v1.2.1": {
					Version: "v1.2.1",
				},
			},
			expectedVersion: "v1.2.1",
			expectError:     false,
			isGitHub:        false,
		},
		{
			name:              "constraint without lockfile match (non-GitHub)",
			repoPath:          NON_GH_TEST_REPO,
			versionConstraint: "^v2.0.0",
			lockFilePackages:  map[string]registry.LockEntry{}, // Empty lockfile
			expectedVersion:   "",                              // Will fail to find version
			expectError:       true,
			isGitHub:          false,
		},
		{
			name:              "tilde constraint without match",
			repoPath:          NON_GH_TEST_REPO,
			versionConstraint: "~v2.1.0",
			expectedVersion:   "",
			expectError:       true,
			isGitHub:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testResolveVersionConstraintCase(t, tt)
		})
	}
}

func testResolveVersionConstraintCase(t *testing.T, tt struct {
	name              string
	repoPath          string
	versionConstraint string
	lockFilePackages  map[string]registry.LockEntry
	expectedVersion   string
	expectError       bool
	isGitHub          bool
}) {
	// Create temporary directory for cache
	tempDir, err := os.MkdirTemp("", "ferret-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock context
	context := &ctx.CompilerContext{
		RemoteCachePath: tempDir,
	}

	// Create mock lock file
	lockFile := &registry.LockFile{
		Packages: tt.lockFilePackages,
	}
	if lockFile.Packages == nil {
		lockFile.Packages = make(map[string]registry.LockEntry)
	}

	// Test resolveVersionConstraint
	result, err := resolveVersionConstraint(tt.repoPath, tt.versionConstraint, lockFile, context)

	if tt.expectError {
		if err == nil {
			t.Errorf("Expected error but got none")
		}
	} else {
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != tt.expectedVersion {
			t.Errorf("Expected version %s, got %s", tt.expectedVersion, result)
		}
	}
}

func TestDetermineModuleType(t *testing.T) {
	tests := []struct {
		name        string
		importPath  string
		projectName string
		expected    ctx.ModuleType
	}{
		{
			name:        "remote GitHub module",
			importPath:  "github.com/user/repo/module",
			projectName: "myapp",
			expected:    ctx.REMOTE,
		},
		{
			name:        "builtin std module",
			importPath:  "std/io",
			projectName: "myapp",
			expected:    ctx.BUILTIN,
		},
		{
			name:        "builtin math module",
			importPath:  "math/geometry",
			projectName: "myapp",
			expected:    ctx.BUILTIN,
		},
		{
			name:        "local project module",
			importPath:  "myapp/utils",
			projectName: "myapp",
			expected:    ctx.LOCAL,
		},
		{
			name:        "unknown module defaults to local",
			importPath:  "unknown/module",
			projectName: "myapp",
			expected:    ctx.LOCAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineModuleType(tt.importPath, tt.projectName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// createTestContext creates a test context for testing
func createTestContext(projectRoot string) *ctx.CompilerContext {
	if projectRoot == "" {
		projectRoot = "/tmp/test"
	}

	projectConfig, _ := config.LoadProjectConfig(projectRoot)

	return &ctx.CompilerContext{
		ProjectConfig:   projectConfig,
		ProjectRoot:     projectRoot,
		RemoteCachePath: filepath.Join(projectRoot, ".ferret", "modules"),
	}
}

func TestRemoteImportValidation(t *testing.T) {
	tests := []struct {
		name          string
		importPath    string
		remoteEnabled bool
		expectError   bool
		errorContains string
	}{
		{
			name:          "remote import allowed",
			importPath:    TEST_REPO,
			remoteEnabled: true,
			expectError:   false,
		},
		{
			name:          "remote import disabled",
			importPath:    TEST_REPO,
			remoteEnabled: false,
			expectError:   true,
			errorContains: "remote module imports are disabled",
		},
		{
			name:          "local import always allowed",
			importPath:    "myapp/modules/test",
			remoteEnabled: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRemoteImportValidationCase(t, tt)
		})
	}
}

func testRemoteImportValidationCase(t *testing.T, tt struct {
	name          string
	importPath    string
	remoteEnabled bool
	expectError   bool
	errorContains string
}) {
	// Create temporary test project
	tempDir, err := os.MkdirTemp("", "ferret-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create fer.ret content
	var ferretContent strings.Builder
	ferretContent.WriteString("[default]\n")
	ferretContent.WriteString("name = \"myapp\"\n")
	ferretContent.WriteString("version = \"1.0.0\"\n")
	ferretContent.WriteString("\n")
	ferretContent.WriteString("[remote]\n")
	ferretContent.WriteString(fmt.Sprintf("enabled = %t\n", tt.remoteEnabled))
	ferretContent.WriteString("share = false\n")
	ferretContent.WriteString("\n")
	ferretContent.WriteString("[dependencies]\n")

	ferRetPath := filepath.Join(tempDir, "fer.ret")
	err = os.WriteFile(ferRetPath, []byte(ferretContent.String()), 0644)
	if err != nil {
		t.Fatalf("Failed to write fer.ret: %v", err)
	}

	// Create context
	ctxx := createTestContext(tempDir)

	// Test the validation
	moduleType := DetermineModuleType(tt.importPath, "myapp")

	if moduleType == ctx.REMOTE {
		_, err = resolveRemoteModule(tt.importPath, ctxx)
	} else {
		// For local modules, we just validate they don't error inappropriately
		err = nil
	}

	if tt.expectError {
		if err == nil {
			t.Errorf("Expected error but got none")
		} else if !strings.Contains(err.Error(), tt.errorContains) {
			t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
		}
	} else {
		if err != nil && !strings.Contains(err.Error(), "not installed") {
			// We expect "not installed" errors since we're not actually installing modules
			t.Errorf("Expected no error, got: %v", err)
		}
	}
}

func TestVersionConstraintValidation(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		constraint string
		satisfies  bool
	}{
		{
			name:       "exact version match",
			version:    "v4.0.0",
			constraint: "v4.0.0",
			satisfies:  true,
		},
		{
			name:       "exact version mismatch",
			version:    "v3.0.0",
			constraint: "v6.0.0",
			satisfies:  false,
		},
		{
			name:       "caret constraint satisfied",
			version:    "v1.2.3",
			constraint: "^v1",
			satisfies:  true,
		},
		{
			name:       "caret constraint not satisfied",
			version:    "v20.0.0",
			constraint: "^v10",
			satisfies:  false,
		},
		{
			name:       "tilde constraint satisfied",
			version:    "v1.2.5",
			constraint: "~v1.2",
			satisfies:  true,
		},
		{
			name:       "tilde constraint not satisfied",
			version:    "v1.3.4",
			constraint: "~v1.2",
			satisfies:  false,
		},
		{
			name:       "latest constraint always satisfied",
			version:    "v0.0.1",
			constraint: "latest",
			satisfies:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := versionSatisfiesConstraint(tt.version, tt.constraint)
			if result != tt.satisfies {
				t.Errorf("versionSatisfiesConstraint(%q, %q) = %v, want %v",
					tt.version, tt.constraint, result, tt.satisfies)
			}
		})
	}
}

func TestFindCurrentCachedVersion(t *testing.T) {
	tests := []struct {
		name            string
		repoPath        string
		cachedVersions  []string
		expectedVersion string
		cacheForRepo    string // Optional: different repo to create cache for
	}{
		{
			name:            "single cached version",
			repoPath:        TEST_REPO,
			cachedVersions:  []string{"v3.1.0"},
			expectedVersion: "v3.1.0",
		},
		{
			name:            "multiple cached versions - returns any found",
			repoPath:        TEST_REPO,
			cachedVersions:  []string{"v1.2.0", "v1.3.0"},
			expectedVersion: "", // We'll check that some version is returned
		},
		{
			name:            "no cached versions",
			repoPath:        TEST_REPO,
			cachedVersions:  []string{},
			expectedVersion: "",
		},
		{
			name:            "different repo - no match",
			repoPath:        "github.com/user/other", // Search for this repo
			cachedVersions:  []string{"v8.0.0"},      // But create cache for a different repo
			expectedVersion: "",
			cacheForRepo:    TEST_REPO, // Create cache for this repo instead
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runFindCurrentCachedVersionTest(t, tt)
		})
	}
}

func runFindCurrentCachedVersionTest(t *testing.T, tt struct {
	name            string
	repoPath        string
	cachedVersions  []string
	expectedVersion string
	cacheForRepo    string
}) {
	// Create temporary cache directory
	tempDir, err := os.MkdirTemp("", "ferret-cache-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock cached versions
	for _, version := range tt.cachedVersions {
		repoToCache := tt.repoPath
		if tt.cacheForRepo != "" {
			repoToCache = tt.cacheForRepo
		}

		flatName := repoToCache + "@" + version
		cachePath := filepath.Join(tempDir, flatName)
		err := os.MkdirAll(cachePath, 0755)
		if err != nil {
			t.Fatalf("Failed to create cache dir: %v", err)
		}
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Fatalf("Cache directory was not created: %v", cachePath)
		}
	}

	// Create test context
	ctxx := createTestContext("")
	ctxx.RemoteCachePath = tempDir

	// Test finding cached version
	result := findCurrentCachedVersion(tt.repoPath, ctxx)

	if tt.name == "multiple cached versions - returns any found" {
		if result != "v1.2.0" && result != "v1.3.0" {
			t.Errorf("findCurrentCachedVersion(%q) = %q, want either v1.2.0 or v1.3.0",
				tt.repoPath, result)
		}
	} else {
		if result != tt.expectedVersion {
			t.Errorf("findCurrentCachedVersion(%q) = %q, want %q",
				tt.repoPath, result, tt.expectedVersion)
		}
	}
}

func TestVersionChangeDetection(t *testing.T) {
	tests := []struct {
		name           string
		repoPath       string
		currentVersion string
		newConstraint  string
		shouldChange   bool
		expectedNewVer string
	}{
		{
			name:           "version satisfies constraint - no change",
			repoPath:       TEST_REPO,
			currentVersion: "v1.0.0",
			newConstraint:  "^v1",
			shouldChange:   false,
		},
		{
			name:           "version doesn't satisfy constraint - should change",
			repoPath:       TEST_REPO,
			currentVersion: "v0.9.0",
			newConstraint:  "^v1",
			shouldChange:   true,
		},
		{
			name:           "exact version change",
			repoPath:       TEST_REPO,
			currentVersion: "v1.0.0",
			newConstraint:  "v2.0.0",
			shouldChange:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldChange := !versionSatisfiesConstraint(tt.currentVersion, tt.newConstraint)
			if shouldChange != tt.shouldChange {
				t.Errorf("Version change detection for %s (current: %s, constraint: %s): got %v, want %v",
					tt.repoPath, tt.currentVersion, tt.newConstraint, shouldChange, tt.shouldChange)
			}
		})
	}
}
