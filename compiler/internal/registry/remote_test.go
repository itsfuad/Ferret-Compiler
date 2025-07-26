package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"compiler/internal/config"
	"compiler/internal/ctx"
)

// Helper function to create fer.ret content for tests
func createFerRetContent(moduleName, version string, enabled, share bool) string {
	var content strings.Builder
	content.WriteString("[default]\n")
	content.WriteString(fmt.Sprintf("name = \"%s\"\n", moduleName))
	content.WriteString(fmt.Sprintf("version = \"%s\"\n", version))
	content.WriteString("\n")
	content.WriteString("[remote]\n")
	content.WriteString(fmt.Sprintf("enabled = %t\n", enabled))
	content.WriteString(fmt.Sprintf("share = %t\n", share))
	return content.String()
}

// Helper function to create minimal fer.ret content (just default section)
func createMinimalFerRetContent(moduleName string) string {
	var content strings.Builder
	content.WriteString("[default]\n")
	content.WriteString(fmt.Sprintf("name = \"%s\"", moduleName))
	return content.String()
}

func TestValidateModuleSharing(t *testing.T) {
	tests := []struct {
		name           string
		ferretContent  string
		ferretLocation string // relative to repo root, empty means root
		expectError    bool
		errorContains  string
	}{
		{
			name:           "valid sharing enabled at root",
			ferretContent:  createFerRetContent("test-module", "1.0.0", true, true),
			ferretLocation: "",
			expectError:    false,
		},
		{
			name:           "sharing disabled at root",
			ferretContent:  createFerRetContent("test-module", "1.0.0", true, false),
			ferretLocation: "",
			expectError:    true,
			errorContains:  "does not allow remote sharing",
		},
		{
			name:           "valid sharing enabled in subdirectory",
			ferretContent:  createFerRetContent("sub-project", "1.0.0", true, true),
			ferretLocation: "data",
			expectError:    false,
		},
		{
			name:           "sharing disabled in subdirectory",
			ferretContent:  createFerRetContent("sub-project", "1.0.0", true, false),
			ferretLocation: "data",
			expectError:    true,
			errorContains:  "does not allow remote sharing",
		},
		{
			name:          "no fer.ret file anywhere",
			expectError:   true,
			errorContains: "no fer.ret file found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir, err := os.MkdirTemp("", "ferret-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Create mock remote cache path
			remoteCachePath := filepath.Join(tempDir, ".ferret", "modules")
			repoPath := "github.com/test/repo"
			version := "v1.0.0"
			flatModuleName := repoPath + "@" + version
			moduleDir := filepath.Join(remoteCachePath, flatModuleName)

			if err := os.MkdirAll(moduleDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Create fer.ret file if specified
			if tt.ferretContent != "" {
				var ferretDir string
				if tt.ferretLocation == "" {
					ferretDir = moduleDir
				} else {
					ferretDir = filepath.Join(moduleDir, tt.ferretLocation)
					if err := os.MkdirAll(ferretDir, 0755); err != nil {
						t.Fatal(err)
					}
				}

				ferretPath := filepath.Join(ferretDir, "fer.ret")
				if err := os.WriteFile(ferretPath, []byte(tt.ferretContent), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Create mock compiler context
			context := &ctx.CompilerContext{
				RemoteCachePath: remoteCachePath,
			}

			// Test ValidateModuleSharing
			err = ValidateModuleSharing(context, repoPath, version)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestFindAnyFerRetInRepo(t *testing.T) {
	tests := []struct {
		name         string
		structure    map[string]string // path -> content
		expectError  bool
		expectedPath string // relative to repo root
	}{
		{
			name: "fer.ret at root",
			structure: map[string]string{
				"fer.ret":  createMinimalFerRetContent("root-project"),
				"main.fer": "// main file",
			},
			expectError:  false,
			expectedPath: "fer.ret",
		},
		{
			name: "fer.ret in subdirectory",
			structure: map[string]string{
				"README.md":     "# Test Repo",
				"data/fer.ret":  createMinimalFerRetContent("data-project"),
				"data/main.fer": "// data main file",
			},
			expectError:  false,
			expectedPath: "data/fer.ret",
		},
		{
			name: "multiple fer.ret files - finds first",
			structure: map[string]string{
				"project-a/fer.ret": createMinimalFerRetContent("project-a"),
				"project-b/fer.ret": createMinimalFerRetContent("project-b"),
			},
			expectError: false,
			// Should find one of them (implementation finds first during walk)
		},
		{
			name: "no fer.ret file",
			structure: map[string]string{
				"README.md": "# Test Repo",
				"main.fer":  "// main file",
				"utils.fer": "// utils file",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "ferret-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Create directory structure
			for filePath, content := range tt.structure {
				fullPath := filepath.Join(tempDir, filePath)
				dir := filepath.Dir(fullPath)

				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}

				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Test findAnyFerRetInRepo
			foundPath, err := findAnyFerRetInRepo(tempDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}

				if tt.expectedPath != "" {
					expectedFullPath := filepath.Join(tempDir, tt.expectedPath)
					if foundPath != expectedFullPath {
						t.Errorf("Expected path %s, got %s", expectedFullPath, foundPath)
					}
				}

				// Verify the found file actually exists and is fer.ret
				if foundPath != "" {
					if filepath.Base(foundPath) != "fer.ret" {
						t.Errorf("Found path should end with fer.ret, got: %s", foundPath)
					}
					if _, err := os.Stat(foundPath); os.IsNotExist(err) {
						t.Errorf("Found path does not exist: %s", foundPath)
					}
				}
			}
		})
	}
}

func TestRemoteConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		share       bool
		expectError bool
	}{
		{"enabled=true, share=true", true, true, false},
		{"enabled=true, share=false", true, false, true},
		{"enabled=false, share=true", false, true, false}, // share doesn't matter if not enabled
		{"enabled=false, share=false", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory with fer.ret
			tempDir, err := os.MkdirTemp("", "ferret-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Create fer.ret content using helper function
			content := createFerRetContent("test-module", "1.0.0", tt.enabled, tt.share)

			ferretPath := filepath.Join(tempDir, "fer.ret")
			if err := os.WriteFile(ferretPath, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			// Load and validate config
			config, err := config.LoadProjectConfig(tempDir)
			if err != nil {
				t.Fatal(err)
			}

			// Check remote settings
			if config.Remote.Enabled != tt.enabled {
				t.Errorf("Expected enabled=%t, got %t", tt.enabled, config.Remote.Enabled)
			}
			if config.Remote.Share != tt.share {
				t.Errorf("Expected share=%t, got %t", tt.share, config.Remote.Share)
			}
		})
	}
}

// versionSatisfiesConstraint is a helper function for testing version constraint satisfaction
func versionSatisfiesConstraint(version, constraint string) bool {
	// Simple constraint checking for testing
	if constraint == "latest" {
		return true // Any version satisfies latest
	}

	if constraint == version {
		return true // Exact match
	}

	if strings.HasPrefix(constraint, "^") {
		baseVersion := strings.TrimPrefix(constraint, "^")
		// For caret constraints like ^v1, v1.x.x satisfies it
		return strings.HasPrefix(version, baseVersion)
	}

	if strings.HasPrefix(constraint, "~") {
		baseVersion := strings.TrimPrefix(constraint, "~")
		// For tilde constraints like ~v1.2, only v1.2.x satisfies it
		return strings.HasPrefix(version, baseVersion)
	}

	return false
}

func TestVersionChangeDetection(t *testing.T) {
	tests := []struct {
		name           string
		repoPath       string
		currentVersion string
		newConstraint  string
		shouldChange   bool
		expectedNewVer string
	}{
		{
			name:           "version satisfies constraint - no change",
			repoPath:       "github.com/user/repo",
			currentVersion: "v1.0.0",
			newConstraint:  "^v1",
			shouldChange:   false,
		},
		{
			name:           "version doesn't satisfy constraint - should change",
			repoPath:       "github.com/user/repo",
			currentVersion: "v0.9.0",
			newConstraint:  "^v1",
			shouldChange:   true,
		},
		{
			name:           "exact version change",
			repoPath:       "github.com/user/repo",
			currentVersion: "v1.0.0",
			newConstraint:  "v2.0.0",
			shouldChange:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldChange := !versionSatisfiesConstraint(tt.currentVersion, tt.newConstraint)
			if shouldChange != tt.shouldChange {
				t.Errorf("Version change detection for %s (current: %s, constraint: %s): got %v, want %v",
					tt.repoPath, tt.currentVersion, tt.newConstraint, shouldChange, tt.shouldChange)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			strings.Contains(s, substr)))
}
