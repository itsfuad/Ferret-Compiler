package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Simple integration test for the init functionality
func TestInitCommand(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Build the binary for testing
	binaryName := "ferret_test"
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		binaryName += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", binaryName, "./cmd")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove(binaryName)

	// Test init command in temporary directory
	cmd := exec.Command("./"+binaryName, "init", tempDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Init command failed: %v, output: %s", err, output)
	}

	// Check that config file was created
	configPath := filepath.Join(tempDir, ".ferret.json")
	if _, err := os.Stat(filepath.FromSlash(configPath)); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Check output contains success message
	outputStr := string(output)
	if !strings.Contains(outputStr, "Project configuration initialized") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
}

// Test help message
func TestHelpMessage(t *testing.T) {
	// Build the binary for testing
	binaryName := "ferret_test_help"
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		binaryName += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", binaryName, "./cmd")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}
	defer os.Remove(binaryName)

	// Test help command (no arguments)
	cmd := exec.Command("./" + binaryName)
	output, err := cmd.CombinedOutput()

	// Should exit with code 1 and show usage
	if err == nil {
		t.Error("Expected command to fail with exit code 1")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage: ferret") {
		t.Errorf("Expected usage message, got: %s", outputStr)
	}
}
