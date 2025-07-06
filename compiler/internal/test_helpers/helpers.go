package test_helpers

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTestFile creates a temporary test file and returns its path
func CreateTestFile(t *testing.T) string {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.fer")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()
	return filePath
}

func CreateTestFileWithContent(t *testing.T, content string) string {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.fer")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}
