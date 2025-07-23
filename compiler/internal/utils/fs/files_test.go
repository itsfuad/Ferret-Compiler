package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"compiler/internal/config"
	"compiler/internal/ctx"
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
		{"Built-in std module", "testproject", "std/io", true},         // Should error with "not implemented yet"
		{"Built-in math module", "testproject", "math/geometry", true}, // Should error with "not implemented yet"
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
