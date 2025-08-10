package cli

import (
	"os"
	"path/filepath"
	"testing"

	"ferret/cmd/flags"
)

// Integration test for the init functionality
func TestInitFunctionality(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save original os.Args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test init in temporary directory
	os.Args = []string{"ferret", "init", tempDir}

	args := flags.ParseArgs()

	if !args.InitProject {
		t.Fatal("Expected initProject to be true")
	}
	if args.ProjectName != tempDir {
		t.Errorf("Expected initPath to be %s, got %s", tempDir, args.ProjectName)
	}
	if args.Filename != "" {
		t.Errorf("Expected filename to be empty, got %s", args.Filename)
	}
	if args.Debug {
		t.Error("Expected debug to be false")
	}
	if args.OutputPath != "" {
		t.Errorf("Expected outputPath to be empty, got %s", args.OutputPath)
	}

	// Verify the config file path would be correct
	expectedConfigPath := filepath.Join(tempDir, "fer.ret")
	if _, err := os.Stat(filepath.FromSlash(expectedConfigPath)); err == nil {
		t.Error("Config file should not exist yet (we only parsed args)")
	}
}

func TestIsInProjectRoot(t *testing.T) {
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Should be false when no fer.ret
	os.Chdir(tempDir)
	if isInProjectRoot() {
		t.Error("Expected false when fer.ret is missing")
	}

	// Create fer.ret
	f, err := os.Create(filepath.Join(tempDir, "fer.ret"))
	if err != nil {
		t.Fatalf("Failed to create fer.ret: %v", err)
	}
	f.Close()

	if !isInProjectRoot() {
		t.Error("Expected true when fer.ret is present")
	}
}
