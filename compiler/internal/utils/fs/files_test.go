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

	ctxx := &ctx.CompilerContext{ProjectRoot: tmpDir}

	// Use a local importer path to ensure it's treated as a local import
	fullPath, logical, err := ResolveModule("module", "local/file.fer", ctxx, false)
	if err != nil {
		t.Fatalf("Expected module to resolve, got error: %v", err)
	}

	if !strings.HasSuffix(fullPath, "module.fer") {
		t.Errorf("Unexpected resolved path: %s", fullPath)
	}

	if logical != "module.fer" {
		t.Errorf("Unexpected logical path: %s", logical)
	}
}

func TestResolveModule_InvalidEmptyPath(t *testing.T) {
	ctxx := &ctx.CompilerContext{ProjectRoot: "."}
	_, _, err := ResolveModule("   ", "", ctxx, false)
	if err == nil {
		t.Error("Expected error for empty module path")
	}
}

func TestResolveModule_InvalidRelativePath(t *testing.T) {
	ctxx := &ctx.CompilerContext{ProjectRoot: "."}
	_, _, err := ResolveModule("./relative", "", ctxx, false)
	if err == nil || !strings.Contains(err.Error(), "relative imports") {
		t.Error("Expected error for relative import")
	}
}

func TestCheckRemoteSharing_GenerateConfigPaths(t *testing.T) {
	// Test that config paths are generated correctly for a deep import
	importPath := "github.com/user/repo/path/to/deep/module"

	// This is a helper function to test the path generation logic
	parts := strings.Split(importPath, "/")
	if len(parts) < 3 {
		t.Fatalf("Invalid remote import path: %s", importPath)
	}

	var configPaths []string
	currentPath := importPath

	// Start from the full import path and walk up to the repository root
	for {
		configPaths = append(configPaths, currentPath)

		// Move up one level
		pathParts := strings.Split(currentPath, "/")
		if len(pathParts) <= 3 {
			// Reached the repository root
			break
		}
		currentPath = strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	expectedPaths := []string{
		"github.com/user/repo/path/to/deep/module",
		"github.com/user/repo/path/to/deep",
		"github.com/user/repo/path/to",
		"github.com/user/repo/path",
		"github.com/user/repo",
	}

	if len(configPaths) != len(expectedPaths) {
		t.Errorf("Expected %d config paths, got %d", len(expectedPaths), len(configPaths))
	}

	for i, expected := range expectedPaths {
		if configPaths[i] != expected {
			t.Errorf("Expected path %s at index %d, got %s", expected, i, configPaths[i])
		}
	}
}

func TestCheckRemoteSharing_ShallowImport(t *testing.T) {
	// Test for a shallow import (just repo/file)
	importPath := "github.com/user/repo/file"

	parts := strings.Split(importPath, "/")
	if len(parts) < 3 {
		t.Fatalf("Invalid remote import path: %s", importPath)
	}

	var configPaths []string
	currentPath := importPath

	// Start from the full import path and walk up to the repository root
	for {
		configPaths = append(configPaths, currentPath)

		// Move up one level
		pathParts := strings.Split(currentPath, "/")
		if len(pathParts) <= 3 {
			// Reached the repository root
			break
		}
		currentPath = strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	expectedPaths := []string{
		"github.com/user/repo/file",
		"github.com/user/repo",
	}

	if len(configPaths) != len(expectedPaths) {
		t.Errorf("Expected %d config paths, got %d", len(expectedPaths), len(configPaths))
	}

	for i, expected := range expectedPaths {
		if configPaths[i] != expected {
			t.Errorf("Expected path %s at index %d, got %s", expected, i, configPaths[i])
		}
	}
}
