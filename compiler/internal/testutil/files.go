package testutil

import (
	"compiler/constants"
	"os"
	"path/filepath"
	"testing"
)

// CreateTempProject creates a temporary directory structure for testing
func CreateTempProject(t *testing.T) string {
	tempDir := t.TempDir()

	// skip if fer.ret file already exists
	if _, err := os.Stat(filepath.Join(tempDir, constants.CONFIG_FILE)); err == nil {
		return tempDir
	}

	// create a fer.ret file
	if err := os.WriteFile(filepath.Join(tempDir, constants.CONFIG_FILE), []byte("name = \"demo-apps\""), 0644); err != nil {
		t.Fatalf("Failed to create fer.ret file: %v", err)
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
