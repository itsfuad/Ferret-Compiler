package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Test help message
func TestHelpMessage(t *testing.T) {
	// Build the binary for testing
	binaryName := "ferret_test_help"
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		binaryName += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", binaryName, ".")
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
	if !strings.Contains(outputStr, "Ferret") {
		t.Errorf("Expected usage message, got: %s", outputStr)
	}
}
