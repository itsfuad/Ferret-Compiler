package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir", "nested")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a ferret.project.json in the root
	config := &ProjectConfig{
		Compiler: CompilerConfig{
			Version: "0.1.0",
		},
		Cache: CacheConfig{
			Path: ".ferret/modules",
		},
		Remote: RemoteConfig{
			Enabled: true,
			Share:   false,
		},
	}

	configPath := filepath.Join(tmpDir, "ferret.project.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test loading from nested directory
	loadedConfig, projectRoot, err := LoadProjectConfig(subDir)
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	if projectRoot != tmpDir {
		t.Errorf("Expected project root %s, got %s", tmpDir, projectRoot)
	}

	if loadedConfig.Compiler.Version != "0.1.0" {
		t.Errorf("Expected version 0.1.0, got %s", loadedConfig.Compiler.Version)
	}

	if !loadedConfig.Remote.Enabled {
		t.Error("Expected remote enabled to be true")
	}

	if loadedConfig.Remote.Share {
		t.Error("Expected remote share to be false")
	}
}

func TestLoadProjectConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := LoadProjectConfig(tmpDir)
	if err == nil {
		t.Error("Expected error when no ferret.project.json is found")
	}
}

func TestLoadProjectConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ferret.project.json")

	// Write invalid JSON
	if err := os.WriteFile(configPath, []byte(`{ invalid json }`), 0644); err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	_, _, err := LoadProjectConfig(tmpDir)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadProjectConfig_MissingVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ferret.project.json")

	// Write config without version
	invalidConfig := `{
		"compiler": {
			"cache": {
				"path": ".ferret/modules"
			},
			"remote": {
				"enabled": true,
				"share": false
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, _, err := LoadProjectConfig(tmpDir)
	if err == nil {
		t.Error("Expected error for missing version")
	}
}

func TestCreateDefaultProjectConfig(t *testing.T) {
	tmpDir := t.TempDir()

	if err := CreateDefaultProjectConfig(tmpDir); err != nil {
		t.Fatalf("Failed to create default config: %v", err)
	}

	// Verify the file was created
	configPath := filepath.Join(tmpDir, "ferret.project.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify the config
	config, _, err := LoadProjectConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load created config: %v", err)
	}

	if config.Compiler.Version != "0.1.0" {
		t.Errorf("Expected default version 0.1.0, got %s", config.Compiler.Version)
	}

	if config.Cache.Path != ".ferret/modules" {
		t.Errorf("Expected default cache path .ferret/modules, got %s", config.Cache.Path)
	}

	if !config.Remote.Enabled {
		t.Error("Expected default remote enabled to be true")
	}

	if config.Remote.Share {
		t.Error("Expected default remote share to be false")
	}
}

func TestIsProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()

	// Should return false when no config exists
	if IsProjectRoot(tmpDir) {
		t.Error("Expected IsProjectRoot to return false when no config exists")
	}

	// Create a config file
	if err := CreateDefaultProjectConfig(tmpDir); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Should return true when config exists
	if !IsProjectRoot(tmpDir) {
		t.Error("Expected IsProjectRoot to return true when config exists")
	}
}
