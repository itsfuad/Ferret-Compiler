package cli

import (
	"os"
	"path/filepath"
	"testing"

	"compiler/cmd/flags"
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
	if args.Debug {
		t.Error("Expected debug to be false")
	}

	// Verify the config file path would be correct
	expectedConfigPath := filepath.Join(tempDir, "fer.ret")
	if _, err := os.Stat(filepath.FromSlash(expectedConfigPath)); err == nil {
		t.Error("Config file should not exist yet (we only parsed args)")
	}
}

// Test for run command functionality
func TestRunCommandParsing(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test run command with debug flag
	os.Args = []string{"ferret", "run", "--debug"}

	args := flags.ParseArgs()

	if !args.RunCommand {
		t.Fatal("Expected RunCommand to be true")
	}
	if !args.Debug {
		t.Error("Expected debug to be true")
	}
}
