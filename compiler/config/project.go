package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"compiler/cmd/flags"
	"compiler/colors"
	"compiler/constants"
	"compiler/toml"
)

const SHAREKEY = "allow-sharing"
const REMOTEKEY = "allow-remote-import"
const EXTERNALKEY = "allow-neighbor-import"

// ProjectConfig represents the structure
type ProjectConfig struct {
	Name         string           `toml:"name"`
	Compiler     CompilerConfig   `toml:"compiler"`
	Build        BuildConfig      `toml:"build"`
	Cache        CacheConfig      `toml:"cache"`
	External     ExternalConfig   `toml:"external"`
	Neighbors    NeighborConfig   `toml:"neighbors"`
	Dependencies DependencyConfig `toml:"dependencies"`
	// Top-level project metadata
	ProjectRoot string
}

var defaultConfig = toml.TOMLData{
	"default": toml.TOMLTable{
		"name": "",
	},
	"compiler": toml.TOMLTable{
		"version": "",
	},
	"build": toml.TOMLTable{
		"entry":  "",
		"output": "",
	},
	"cache": toml.TOMLTable{
		"path": "",
	},
	"external": toml.TOMLTable{
		SHAREKEY:    "",
		REMOTEKEY:   "",
		EXTERNALKEY: "",
	},
	"neighbors":    toml.TOMLTable{},
	"dependencies": toml.TOMLTable{},
}

func (conf *ProjectConfig) Save() {

	// Validate the configuration
	tomData := defaultConfig
	tomData["default"] = toml.TOMLTable{
		"name": conf.Name,
	}
	tomData["compiler"] = toml.TOMLTable{
		"version": conf.Compiler.Version,
	}
	tomData["build"] = toml.TOMLTable{
		"entry":  conf.Build.Entry,
		"output": conf.Build.Output,
	}
	tomData["cache"] = toml.TOMLTable{
		"path": conf.Cache.Path,
	}
	tomData["external"] = toml.TOMLTable{
		SHAREKEY:    conf.External.AllowSharing,
		REMOTEKEY:   conf.External.AllowRemoteImport,
		EXTERNALKEY: conf.External.AllowExternalImport,
	}

	for key, value := range conf.Neighbors.Projects {
		tomData["neighbors"][key] = value
	}
	for key, value := range conf.Dependencies.Modules {
		tomData["dependencies"][key] = value
	}

	// Save the configuration to the fer.ret file
	if err := toml.WriteTOMLFile(filepath.Join(conf.ProjectRoot, constants.CONFIG_FILE), tomData, nil); err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}
}

// CompilerConfig contains compiler-specific settings
type CompilerConfig struct {
	Version string `toml:"version"`
}

// CacheConfig defines cache settings
type CacheConfig struct {
	Path string `toml:"path"`
}

type ExternalConfig struct {
	AllowSharing        bool `toml:"allow-sharing"`
	AllowRemoteImport   bool `toml:"allow-remote-import"`
	AllowExternalImport bool `toml:"allow-neighbor-import"`
}

// BuildConfig defines build settings
type BuildConfig struct {
	Entry  string `toml:"entry"`  // entrypoint file
	Output string `toml:"output"` // optional explicit output path
}

type DependencyConfig struct {
	Modules map[string]string `toml:"dependencies"` // module_name -> version
}

// NeighborConfig defines neighboring project mappings (like Go's replace directive)
type NeighborConfig struct {
	Projects map[string]string `toml:"neighbors"` // project_name -> local_path
}

// CreateDefaultProjectConfig creates a default fer.ret configuration file
func CreateDefaultProjectConfig(projectName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}

	configPath := filepath.Join(cwd, constants.CONFIG_FILE)

	// Generate config using TOML data structure for consistency
	configData := generateDefaultConfigData(projectName)

	if err := toml.WriteTOMLFile(configPath, configData, nil); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	colors.GREEN.Printf("📁 Created %s successfully!\n", constants.CONFIG_FILE)
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
			fmt.Printf("Invalid input %q. Please enter true/false, yes/no, or y/n: ", value)
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
		fmt.Println("⚠️  Project name must not contain spaces or special characters.")
		os.Exit(1)
	}

	allowShare, err := ReadBoolFromPrompt("Allow sharing of this project with other projects? (true/false, default: false): ", false)
	if err != nil {
		fmt.Printf("❌ Error reading share setting: %v\n", err)
		os.Exit(1)
	}

	allowRemoteImport, err := ReadBoolFromPrompt("Allow remote imports? [e.g. github, gitlab] (true/false, default: false): ", false)
	if err != nil {
		fmt.Printf("❌ Error reading remote import setting: %v\n", err)
		os.Exit(1)
	}

	allowNeighborImport, err := ReadBoolFromPrompt("Allow neighbor imports from other projects? (true/false, default: false): ", false)
	if err != nil {
		fmt.Printf("❌ Error reading neighbor import setting: %v\n", err)
		os.Exit(1)
	}

	configData := defaultConfig

	configData["default"] = toml.TOMLTable{
		"name": projectName,
	}

	configData["compiler"] = toml.TOMLTable{
		"version": flags.FERRET_VERSION,
	}

	configData["build"] = toml.TOMLTable{
		"entry":  "yourproject.fer",
		"output": "bin/" + projectName,
	}

	configData["cache"] = toml.TOMLTable{
		"path": ".ferret",
	}

	configData["external"] = toml.TOMLTable{
		SHAREKEY:    allowShare,
		REMOTEKEY:   allowRemoteImport,
		EXTERNALKEY: allowNeighborImport,
	}

	configData["neighbors"] = toml.TOMLTable{}

	configData["dependencies"] = toml.TOMLTable{}

	return configData
}

func ValidateProjectConfig(config *ProjectConfig) error {
	if config == nil {
		return fmt.Errorf("⚠️  project configuration is nil")
	}

	if config.Name == "" {
		return fmt.Errorf("⚠️  project name is required in the configuration file")
	}

	if config.Compiler.Version == "" {
		return fmt.Errorf("⚠️  compiler version is required in the configuration file")
	}

	if config.Build.Entry == "" {
		return fmt.Errorf("⚠️  build entry point is required in the configuration file")
	}

	if config.Cache.Path == "" {
		return fmt.Errorf("⚠️  cache path is required in the configuration file")
	}

	if config.ProjectRoot == "" {
		return fmt.Errorf("⚠️  project root is required in the configuration file")
	}

	return nil
}

// IsProjectRoot checks if the given directory contains a fer.ret file
func IsProjectRoot(dir string) bool {
	configPath := filepath.Join(dir, constants.CONFIG_FILE)
	_, err := os.Stat(filepath.FromSlash(configPath))
	return err == nil
}

func GetProjectRoot(moduleFullPath string) (string, error) {

	moduleFullPath, err := filepath.Abs(moduleFullPath)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(moduleFullPath)

	// Walk up the directory tree until we find a fer.ret file
	for {
		configPath := filepath.Join(dir, constants.CONFIG_FILE)
		if _, err := os.Stat(filepath.FromSlash(configPath)); err == nil {
			return filepath.ToSlash(dir), nil // Found the project root
		}

		// Move up to parent directory
		parent := filepath.Dir(dir)

		if parent == dir { // Stop if we can't go up further (reached filesystem root)
			break
		}

		dir = parent
	}

	return "", fmt.Errorf("project root not found")
}

func LoadProjectConfig(projectRoot string) (*ProjectConfig, error) {

	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		colors.RED.Printf("❌ Failed to get absolute path of project root: %s\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(projectRoot, constants.CONFIG_FILE)

	// if not exists, ask user to create one
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		colors.RED.Printf("❌ Configuration file %s not found in %s\nCreate project by running 'ferret init' command\n", constants.CONFIG_FILE, projectRoot)
	}

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
	parseExternalSection(tomlData, config)
	parseDependenciesSection(tomlData, config)
	parseNeighborSection(tomlData, config)

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

func parseExternalSection(tomlData toml.TOMLData, config *ProjectConfig) {
	if externalSection, exists := tomlData["external"]; exists {
		if allowRemote, ok := externalSection[REMOTEKEY].(bool); ok {
			config.External.AllowRemoteImport = allowRemote
		}
		if allowSharing, ok := externalSection[SHAREKEY].(bool); ok {
			config.External.AllowSharing = allowSharing
		}
		if allowExternal, ok := externalSection[EXTERNALKEY].(bool); ok {
			config.External.AllowExternalImport = allowExternal
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

func parseNeighborSection(tomlData toml.TOMLData, config *ProjectConfig) {
	if neighborSection, exists := tomlData["neighbors"]; exists {
		config.Neighbors.Projects = make(map[string]string)
		for key, value := range neighborSection {
			if strValue, ok := value.(string); ok {
				config.Neighbors.Projects[key] = strValue
			}
		}
	}
}

func FindProjectRoot(filePath string) (string, error) {

	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("❌ failed to get absolute path of entry: %w", err)
	}

	dir := filepath.Dir(filePath)

	// Walk up the directory tree until we find a fer.ret file
	for {
		configPath := filepath.Join(dir, constants.CONFIG_FILE)
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

	return "", fmt.Errorf("❌ %s not found (searched from: %s up to filesystem root)", constants.CONFIG_FILE, dir)
}
