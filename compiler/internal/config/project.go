package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ferret/colors"
	"ferret/toml"
)

const CONFIG_FILE = "fer.ret"

// ProjectConfig represents the structure
type ProjectConfig struct {
	Compiler     CompilerConfig   `toml:"compiler"`
	Cache        CacheConfig      `toml:"cache"`
	Remote       RemoteConfig     `toml:"remote"`
	Build        BuildConfig      `toml:"build"`
	Dependencies DependencyConfig `toml:"dependencies"`
	ProjectRoot  string

	// Top-level project metadata
	Name string `toml:"name"`
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

// BuildConfig defines build settings
type BuildConfig struct {
	Entry  string `toml:"entry"`  // entrypoint file
	Output string `toml:"output"` // optional explicit output path
}

type DependencyConfig struct {
	Modules map[string]string `toml:"dependencies"` // module_name -> version
}

// CreateDefaultProjectConfig creates a default fer.ret configuration file
func CreateDefaultProjectConfig(projectName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}

	configPath := filepath.Join(cwd, CONFIG_FILE)

	// Generate config using TOML data structure for consistency
	configData := generateDefaultConfigData(projectName)

	if err := toml.WriteTOMLFile(configPath, configData, nil); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	colors.GREEN.Printf("📁 Created %s successfully!\n", CONFIG_FILE)
	return nil
}

func ReadFromPrompt(prompt string, defaultValue string) (string, error) {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		value := strings.TrimSpace(scanner.Text())
		if value == "" {
			return defaultValue, nil // Use default if empty input
		}
		return value, nil // Use provided input
	}
	if err := scanner.Err(); err != nil {
		return defaultValue, fmt.Errorf("error reading input: %w", err)
	}
	return defaultValue, nil
}

// ReadBoolFromPrompt reads a boolean value from user input with validation
func ReadBoolFromPrompt(prompt string, defaultValue bool) (bool, error) {
	defaultStr := "false"
	if defaultValue {
		defaultStr = "true"
	}

	for {
		value, err := ReadFromPrompt(prompt, defaultStr)
		if err != nil {
			return defaultValue, err
		}

		switch strings.ToLower(value) {
		case "true", "yes", "y":
			return true, nil
		case "false", "no", "n":
			return false, nil
		default:
			fmt.Printf("Invalid input '%s'. Please enter true/false, yes/no, or y/n: ", value)
			continue
		}
	}
}

// generateDefaultConfigData creates the default configuration data structure using TOML
func generateDefaultConfigData(projectName string) toml.TOMLData {
	if projectName == "" {
		// Get project name
		name, err := ReadFromPrompt("Enter project name (press enter for default: app): ", "app")
		if err != nil {
			fmt.Printf("❌ Error reading project name: %v\n", err)
			os.Exit(1)
		}
		projectName = name
	}

	//must not contain spaces or special characters in the middle
	if strings.ContainsAny(projectName, " \t\n\r") || strings.ContainsAny(projectName, "!@#$%^&*()+=[]{}|;:'\",.<>?/\\") {
		fmt.Println("ℹ️ Project name must not contain spaces or special characters.")
		os.Exit(1)
	}

	// Get remote enabled setting
	remoteEnabled, err := ReadBoolFromPrompt("Do you want to allow remote module import ([Yes|No|Y|N] default: no)? ", false)
	if err != nil {
		fmt.Printf("❌ Error reading remote setting: %v\n", err)
		os.Exit(1)
	}

	// Get share enabled setting
	shareEnabled, err := ReadBoolFromPrompt("Do you want to allow sharing your modules to others as remote modules ([Yes|No|Y|N] default: no)? ", false)
	if err != nil {
		fmt.Printf("❌ Error reading share setting: %v\n", err)
		os.Exit(1)
	}

	// Build configuration data structure
	configData := toml.TOMLData{
		"default": toml.TOMLTable{
			"name": projectName,
		},
		"compiler": toml.TOMLTable{
			"version": "0.1.0",
		},
		"build": toml.TOMLTable{
			"entry":  "src/main.fer",
			"output": "bin/" + projectName,
		},
		"cache": toml.TOMLTable{
			"path": ".ferret/cache",
		},
		"remote": toml.TOMLTable{
			"enabled": remoteEnabled,
			"share":   shareEnabled,
		},
		"dependencies": toml.TOMLTable{},
	}

	return configData
}

func ValidateProjectConfig(config *ProjectConfig) error {
	if config == nil {
		return fmt.Errorf("ℹ️ project configuration is nil")
	}

	if config.Name == "" {
		return fmt.Errorf("ℹ️ project name is required in the configuration file")
	}

	if config.Compiler.Version == "" {
		return fmt.Errorf("ℹ️ compiler version is required in the configuration file")
	}

	if config.Build.Entry == "" {
		return fmt.Errorf("ℹ️ build entry point is required in the configuration file")
	}

	if config.Cache.Path == "" {
		return fmt.Errorf("ℹ️ cache path is required in the configuration file")
	}

	if config.ProjectRoot == "" {
		return fmt.Errorf("ℹ️ project root is required in the configuration file")
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
		return nil, fmt.Errorf("❌ failed to parse config file: %w", err)
	}

	config := &ProjectConfig{
		ProjectRoot: projectRoot,
	}

	// Parse each section
	parseDefaultSection(tomlData, config)
	parseCompilerSection(tomlData, config)
	parseBuildSection(tomlData, config)
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

func parseBuildSection(tomlData toml.TOMLData, config *ProjectConfig) {
	if buildSection, exists := tomlData["build"]; exists {
		if entry, ok := buildSection["entry"].(string); ok {
			config.Build.Entry = entry
		}
		if output, ok := buildSection["output"].(string); ok {
			config.Build.Output = output
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
	// Get the absolute path of the entry file
	absEntryFile, err := filepath.Abs(entryFile)
	if err != nil {
		return "", fmt.Errorf("❌ failed to get absolute path of entry file: %w", err)
	}

	// Start from the directory containing the entry file
	dir := filepath.Dir(absEntryFile)
	originalDir := dir // Store original for better error message

	// Walk up the directory tree until we find a fer.ret file
	for {
		configPath := filepath.Join(dir, CONFIG_FILE)

		// Check if fer.ret exists in this directory
		if _, err := os.Stat(configPath); err == nil {
			// Found the project root
			return filepath.ToSlash(dir), nil
		}

		// Move up to parent directory
		parent := filepath.Dir(dir)

		// Stop if we can't go up further (reached filesystem root)
		if parent == dir {
			break
		}

		dir = parent
	}

	return "", fmt.Errorf("❌ %s not found (searched from: %s up to filesystem root)", CONFIG_FILE, originalDir)
}
