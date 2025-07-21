package config

import (
	"fmt"
	"os"
	"path/filepath"

	"ferret/compiler/internal/toml"
)

const CONFIG_FILE = "fer.ret"

// ProjectConfig represents the structure
type ProjectConfig struct {
	Compiler     CompilerConfig   `toml:"compiler"`
	Cache        CacheConfig      `toml:"cache"`
	Remote       RemoteConfig     `toml:"remote"`
	Dependencies DependencyConfig `toml:"dependencies"`
	ProjectRoot  string

	// Top-level project metadata
	Name     string `toml:"name"`
	Version  string `toml:"version"`
	Optimize bool   `toml:"optimize"`
}

// CompilerConfig contains compiler-specific settings
type CompilerConfig struct {
	Version string `toml:"version"`
}

// CacheConfig defines cache settings
type CacheConfig struct {
	Path string `toml:"path"`
}

// RemoteConfig defines remote module import/export settings
type RemoteConfig struct {
	Enabled bool `toml:"enabled"`
	Share   bool `toml:"share"`
}

type DependencyConfig struct {
	Modules map[string]string `toml:"dependencies"` // module_name -> version
}

// CreateDefaultProjectConfig creates a default fer.ret configuration file
func CreateDefaultProjectConfig(projectRoot string) error {
	configPath := filepath.Join(projectRoot, CONFIG_FILE)

	// Create TOML content as a string (simpler than marshalling structs to TOML)
	defaultContent := `name = "ferret_project"
version = "0.1.0"
optimize = false

[compiler]
version = "0.1.0"

[cache]
path = ".ferret/modules"

[remote]
enabled = true
share = false

[dependencies]
# Add your dependencies here
# example_lib = "1.0.0"
`

	if err := os.WriteFile(configPath, []byte(defaultContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsProjectRoot checks if the given directory contains a fer.ret file
func IsProjectRoot(dir string) bool {
	configPath := filepath.Join(dir, CONFIG_FILE)
	_, err := os.Stat(filepath.FromSlash(configPath))
	return err == nil
}

func LoadProjectConfig(projectRoot string) (*ProjectConfig, error) {
	configPath := filepath.Join(projectRoot, CONFIG_FILE)

	// Use our custom TOML parser
	tomlData, err := toml.ParseTOMLFile(filepath.FromSlash(configPath))
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config := &ProjectConfig{
		ProjectRoot: projectRoot,
	}

	// Parse each section
	parseDefaultSection(tomlData, config)
	parseCompilerSection(tomlData, config)
	parseCacheSection(tomlData, config)
	parseRemoteSection(tomlData, config)
	parseDependenciesSection(tomlData, config)

	return config, nil
}

// Helper functions to parse each section
func parseDefaultSection(tomlData toml.TOMLData, config *ProjectConfig) {
	if defaultSection, exists := tomlData["default"]; exists {
		if name, ok := defaultSection["name"].(string); ok {
			config.Name = name
		}
		if version, ok := defaultSection["version"].(string); ok {
			config.Version = version
		}
		if optimize, ok := defaultSection["optimize"].(bool); ok {
			config.Optimize = optimize
		}
	}
}

func parseCompilerSection(tomlData toml.TOMLData, config *ProjectConfig) {
	if compilerSection, exists := tomlData["compiler"]; exists {
		if version, ok := compilerSection["version"].(string); ok {
			config.Compiler.Version = version
		}
	}
}

func parseCacheSection(tomlData toml.TOMLData, config *ProjectConfig) {
	if cacheSection, exists := tomlData["cache"]; exists {
		if path, ok := cacheSection["path"].(string); ok {
			config.Cache.Path = path
		}
	}
}

func parseRemoteSection(tomlData toml.TOMLData, config *ProjectConfig) {
	if remoteSection, exists := tomlData["remote"]; exists {
		if enabled, ok := remoteSection["enabled"].(bool); ok {
			config.Remote.Enabled = enabled
		}
		if share, ok := remoteSection["share"].(bool); ok {
			config.Remote.Share = share
		}
	}
}

func parseDependenciesSection(tomlData toml.TOMLData, config *ProjectConfig) {
	if dependenciesSection, exists := tomlData["dependencies"]; exists {
		config.Dependencies.Modules = make(map[string]string)
		for key, value := range dependenciesSection {
			if strValue, ok := value.(string); ok {
				config.Dependencies.Modules[key] = strValue
			}
		}
	}
}

func FindProjectRoot(entryFile string) (string, error) {
	dir := filepath.Dir(entryFile)
	for {
		configPath := filepath.Join(dir, CONFIG_FILE)
		if _, err := os.Stat(filepath.FromSlash(configPath)); err == nil {
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
