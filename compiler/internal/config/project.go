package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const CONFIG_FILE = ".ferret.json"

// ProjectConfig represents the structure of ferret.project.json
type ProjectConfig struct {
	Compiler     CompilerConfig   `json:"compiler"`
	Cache        CacheConfig      `json:"cache"`
	Remote       RemoteConfig     `json:"remote"`
	Dependencies DependencyConfig `json:"dependencies"`
	ProjectRoot  string
}

// CompilerConfig contains compiler-specific settings
type CompilerConfig struct {
	Version string `json:"version"`
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

type DependencyConfig struct {
	Modules []string `json:"modules,omitempty"`
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
		Dependencies: DependencyConfig{
			Modules: []string{},
		},
	}

	configPath := filepath.Join(projectRoot, CONFIG_FILE)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsProjectRoot checks if the given directory contains a .json file
func IsProjectRoot(dir string) bool {
	configPath := filepath.Join(dir, CONFIG_FILE)
	_, err := os.Stat(configPath)
	return err == nil
}

func LoadProjectConfig(projectRoot string) (*ProjectConfig, error) {
	configPath := filepath.Join(projectRoot, CONFIG_FILE)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config.ProjectRoot = projectRoot
	return &config, nil
}

func FindProjectRoot(entryFile string) (string, error) {
	dir := filepath.Dir(entryFile)
	for {
		configPath := filepath.Join(dir, CONFIG_FILE)
		if _, err := os.Stat(configPath); err == nil {
			return filepath.ToSlash(dir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}
	return "", fmt.Errorf("%s not found", CONFIG_FILE)
}
