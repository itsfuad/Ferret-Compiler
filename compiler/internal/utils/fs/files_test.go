package fs

import (
	"compiler/ctx"
	"os"
	"path/filepath"
	"testing"
)

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
	projectName := "testproject"
	projectDir := filepath.Join(tempDir, projectName)
	err := os.MkdirAll(filepath.Join(projectDir, "module"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test module file
	moduleFile := filepath.Join(projectDir, "module", "test.fer")
	if err := os.WriteFile(moduleFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create context
	ctxx := &ctx.CompilerContext{
		ProjectRoot: projectDir,
	}

	tests := []struct {
		name                string
		importPath          string
		currentFileFullPath string
		wantErr             bool
	}{
		{"Remote import", "github.com/user/repo/module", "", true},
		{"Empty import", "", "", true},
		{"Non-existent local module", "testproject/nonexistent", "", true},
		// Note: Valid local module test would require mocking IsValidFile or setting up more complex file structure
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveModule(tt.importPath, tt.currentFileFullPath, ctxx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveModule(%q, %q, ctx) error = %v, wantErr %v",
					tt.importPath, tt.currentFileFullPath, err, tt.wantErr)
			}
		})
	}
}
