package fs

import (
	"os"
	"testing"
)

func TestIsValidFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Should return true for existing regular file
	if !IsValidFile(tmpFile.Name()) {
		t.Errorf("IsValidFile(%q) = false, want true", tmpFile.Name())
	}

	// Should return false for non-existent file
	if IsValidFile("nonexistent_file_123456789.txt") {
		t.Errorf("IsValidFile(nonexistent_file_123456789.txt) = true, want false")
	}

	// Should return false for a directory
	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	if IsValidFile(tmpDir) {
		t.Errorf("IsValidFile(%q) = true, want false (directory)", tmpDir)
	}
}

func TestFirstPart(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"file.txt", "file"},
		{"dir/file.txt", "dir"},
		{"dir\\file.txt", "dir"},
		{"/root/file.txt", "root"},
		{"\\root\\file.txt", "root"},
		{"dir/subdir/file.txt", "dir"},
		{"dir\\subdir\\file.txt", "dir"},
		{"/", ""},
		{"\\", ""},
		{"\\\\\\\\", ""},
	}

	for _, tt := range tests {
		got := FirstPart(tt.input)
		if got != tt.want {
			t.Errorf("FirstPart(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLastPart(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"file.txt", "file"},
		{"dir/file.txt", "file"},
		{"dir\\file.txt", "file"},
		{"/root/file.txt", "file"},
		{"\\root\\file.txt", "file"},
		{"dir/subdir/file.txt", "file"},
		{"dir\\subdir\\file.txt", "file"},
		{"/", ""},
		{"\\", ""},
		{"\\\\\\\\", ""},
	}

	for _, tt := range tests {
		got := LastPart(tt.input)
		if got != tt.want {
			t.Errorf("LastPart(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
