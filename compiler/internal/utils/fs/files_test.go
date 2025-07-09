package fs

import (
	"compiler/ctx"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile-*.fer")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if !IsValidFile(tmpFile.Name()) {
		t.Errorf("Expected file to be valid: %s", tmpFile.Name())
	}

	if IsValidFile("nonexistent.fer") {
		t.Errorf("Expected non-existent file to be invalid")
	}
}

func TestGitHubPathToRawURL(t *testing.T) {
	url, sub := GitHubPathToRawURL("github.com/user/repo/path/file", "main")
	expected := "https://raw.githubusercontent.com/user/repo/main/path/file.fer"

	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
	if sub != "path/file" {
		t.Errorf("Expected subpath 'path/file', got '%s'", sub)
	}

	invalidURL, _ := GitHubPathToRawURL("invalid.com/user/repo", "main")
	if invalidURL != "" {
		t.Errorf("Expected empty URL for non-GitHub path")
	}
}

func TestResolveProjectRootModule(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "module.fer")
	os.WriteFile(filePath, []byte("test"), 0644)

	ctxx := &ctx.CompilerContext{RootDir: tmpDir}

	absPath, logical, err := ResolveModule("module", "", ctxx, false)
	if err != nil {
		t.Fatalf("Expected module to resolve, got error: %v", err)
	}

	if !strings.HasSuffix(absPath, "module.fer") {
		t.Errorf("Unexpected resolved path: %s", absPath)
	}

	if logical != "module.fer" {
		t.Errorf("Unexpected logical path: %s", logical)
	}
}

func TestResolveModule_InvalidEmptyPath(t *testing.T) {
	ctxx := &ctx.CompilerContext{RootDir: "."}
	_, _, err := ResolveModule("   ", "", ctxx, false)
	if err == nil {
		t.Error("Expected error for empty module path")
	}
}

func TestResolveModule_InvalidRelativePath(t *testing.T) {
	ctxx := &ctx.CompilerContext{RootDir: "."}
	_, _, err := ResolveModule("./relative", "", ctxx, false)
	if err == nil || !strings.Contains(err.Error(), "relative imports") {
		t.Error("Expected error for relative import")
	}
}
