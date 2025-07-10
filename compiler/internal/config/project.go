package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProjectConfig represents the structure of ferret.project.json
type ProjectConfig struct {
	Compiler CompilerConfig `json:"compiler"`
	Cache   CacheConfig  `json:"cache"`
	Remote  RemoteConfig `json:"remote"`
	ProjectRoot string
}

// CompilerConfig contains compiler-specific settings
type CompilerConfig struct {
	Version string       `json:"version"`
}

// CacheConfig defines cache settings
type CacheConfig struct {
	Path string `json:"path"`
}

// RemoteConfig defines remote module import/export settings
type RemoteConfig struct {
	Enabled bool `json:"enabled"`
	Share   bool `json:"share"`
}

// LoadProjectConfig loads a ferret.project.json file from the given directory
// It walks up the directory tree to find the project root
func LoadProjectConfig(startDir string) (*ProjectConfig, string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Walk up the directory tree looking for ferret.project.json
	for {
		configPath := filepath.Join(dir, "ferret.project.json")
		if _, err := os.Stat(configPath); err == nil {
			// Found the config file
			config, err := parseProjectConfig(configPath)
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse %s: %w", configPath, err)
			}
			return config, dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root of the filesystem
			return nil, "", fmt.Errorf("no ferret.project.json found in directory tree starting from %s", startDir)
		}
		dir = parent
	}
}

// parseProjectConfig reads and parses a ferret.project.json file
func parseProjectConfig(configPath string) (*ProjectConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate the configuration
	if err := validateProjectConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validateProjectConfig validates the project configuration
func validateProjectConfig(config *ProjectConfig) error {
	if config.Compiler.Version == "" {
		return fmt.Errorf("compiler version is required")
	}

	if config.Cache.Path == "" {
		// Set default cache path if not specified
		config.Cache.Path = ".ferret/modules"
	}

	return nil
}

// CreateDefaultProjectConfig creates a default ferret.project.json configuration
func CreateDefaultProjectConfig(projectRoot string) error {
	config := &ProjectConfig{
		Compiler: CompilerConfig{
			Version: "0.1.0",
		},
		Cache: CacheConfig{
			Path: ".ferret/modules",
		},
		Remote: RemoteConfig{
			Enabled: true,
			Share:   false, // Default to false for security
		},
	}

	configPath := filepath.Join(projectRoot, "ferret.project.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsProjectRoot checks if the given directory contains a ferret.project.json file
func IsProjectRoot(dir string) bool {
	configPath := filepath.Join(dir, "ferret.project.json")
	_, err := os.Stat(configPath)
	return err == nil
}
