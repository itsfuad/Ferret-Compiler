package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempProject creates a temporary directory structure for testing
func CreateTempProject(t *testing.T) string {
	tempDir := t.TempDir()

	// Create cache directory structure
	cacheDir := filepath.Join(tempDir, ".ferret", "modules")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	return tempDir
}

// CreateTestFile creates a temporary test file with content
func CreateTestFile(t *testing.T, content string) string {
	dir := CreateTempProject(t)
	filePath := filepath.Join(dir, "test.fer")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// CreateTestFileInDir creates a test file in a specific directory
func CreateTestFileInDir(t *testing.T, dir, filename, content string) string {
	// Ensure the target directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}
