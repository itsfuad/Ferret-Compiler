package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"compiler/toml"
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
	Name    string `toml:"name"`
	Version string `toml:"version"`
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

	// Generate config content using strings
	configContent := generateDefaultConfig()

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Created %s successfully!\n", CONFIG_FILE)
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

// generateDefaultConfig creates the default configuration content
func generateDefaultConfig() string {
	var sb strings.Builder

	// Get project name
	projectName, err := ReadFromPrompt("Enter project name (press enter for default: app): ", "app")
	if err != nil {
		fmt.Printf("Error reading project name: %v\n", err)
		os.Exit(1)
	}

	//must not contain spaces or special characters in the middle
	if strings.ContainsAny(projectName, " \t\n\r") || strings.ContainsAny(projectName, "!@#$%^&*()+=[]{}|;:'\",.<>?/\\") {
		fmt.Println("Project name must not contain spaces or special characters.")
		os.Exit(1)
	}

	// Get remote enabled setting
	remoteEnabled, err := ReadBoolFromPrompt("Do you want to allow remote module import ([Yes|No|Y|N] default: no)? ", false)
	if err != nil {
		fmt.Printf("Error reading remote setting: %v\n", err)
		os.Exit(1)
	}

	// Get share enabled setting
	shareEnabled, err := ReadBoolFromPrompt("Do you want to allow sharing your modules to others as remote modules ([Yes|No|Y|N] default: no)? ", false)
	if err != nil {
		fmt.Printf("Error reading share setting: %v\n", err)
		os.Exit(1)
	}

	// Build configuration content
	sb.WriteString("[default]\n")
	sb.WriteString(fmt.Sprintf("name = \"%s\"\n", projectName))
	sb.WriteString("version = \"1.0.0\"\n\n")

	sb.WriteString("[compiler]\n")
	sb.WriteString("version = \"0.1.0\"\n\n")

	sb.WriteString("[remote]\n")
	if remoteEnabled {
		sb.WriteString("enabled = true\n")
	} else {
		sb.WriteString("enabled = false\n")
	}
	if shareEnabled {
		sb.WriteString("share = true\n\n")
	} else {
		sb.WriteString("share = false\n\n")
	}

	sb.WriteString("[dependencies]\n")
	sb.WriteString("# Add your dependencies here\n")
	sb.WriteString("# example = \"1.0.0\"\n")

	return sb.String()
}

func ValidateProjectConfig(config *ProjectConfig) error {
	if config == nil {
		return fmt.Errorf("project configuration is nil")
	}

	if config.Name == "" {
		return fmt.Errorf("project name is required in the configuration file")
	}

	if config.Version == "" {
		return fmt.Errorf("project version is required in the configuration file")
	}

	if config.Compiler.Version == "" {
		return fmt.Errorf("compiler version is required in the configuration file")
	}

	if config.Cache.Path == "" {
		return fmt.Errorf("cache path is required in the configuration file")
	}

	if config.ProjectRoot == "" {
		return fmt.Errorf("project root is required in the configuration file")
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
