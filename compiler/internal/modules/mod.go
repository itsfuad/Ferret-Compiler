package modules

import (
	"compiler/toml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const REMOTE_HOST = "github.com/"

// ModuleType represents the category of a module
type ModuleType int

const (
	UNKNOWN ModuleType = iota
	LOCAL
	BUILTIN
	REMOTE
)

func (mt ModuleType) String() string {
	switch mt {
	case LOCAL:
		return "LOCAL"
	case BUILTIN:
		return "BUILTIN"
	case REMOTE:
		return "REMOTE"
	default:
		return "UNKNOWN"
	}
}

// Built-in modules that are part of the standard library
var BUILTIN_MODULES = map[string]bool{
	"std":  true,
	"math": true,
	"io":   true,
	"os":   true,
	"net":  true,
	"http": true,
	"json": true,
	"time": true,
}

func IsBuiltinModule(importRoot string) bool {
	return BUILTIN_MODULES[importRoot]
}

// GetModuleType categorizes an import path
func GetModuleType(importPath string, projectName string) ModuleType {
	importRoot := FirstPart(importPath)

	if IsRemote(importPath) {
		return REMOTE
	}

	if IsBuiltinModule(importRoot) {
		return BUILTIN
	}

	if importRoot == projectName {
		return LOCAL
	}

	// Default to local for unrecognized paths
	return UNKNOWN
}

func IsRemote(importPath string) bool {
	return strings.HasPrefix(importPath, REMOTE_HOST)
}

// Check if file exists and is a regular file
func IsValidFile(filename string) bool {
	fileInfo, err := os.Stat(filepath.FromSlash(filename))
	return err == nil && fileInfo.Mode().IsRegular()
}

func FirstPart(path string) string {
	if path == "" {
		return ""
	}

	// Handle both forward slashes and backslashes explicitly
	// Replace all backslashes with forward slashes for uniform processing
	normalized := strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(normalized, "/")

	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

func LastPart(path string) string {
	if path == "" {
		return ""
	}

	// Handle both forward slashes and backslashes explicitly
	// Replace all backslashes with forward slashes for uniform processing
	normalized := strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(normalized, "/")

	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}
	return ""
}

// CheckRemoteModuleShareSetting checks if a remote module allows sharing
// by reading its fer.ret configuration file
func CheckRemoteModuleShareSetting(moduleDir string) (bool, error) {
	configPath := filepath.Join(moduleDir, "fer.ret")

	// If no fer.ret file exists, assume sharing is allowed (default behavior)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return true, nil
	}

	// Parse the fer.ret file in the remote module
	tomlData, err := toml.ParseTOMLFile(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to parse fer.ret in remote module: %w", err)
	}

	// Check the [remote] section for share setting
	if remoteSection, exists := tomlData["remote"]; exists {
		if shareValue, ok := remoteSection["share"]; ok {
			if shareBool, ok := shareValue.(bool); ok {
				return shareBool, nil
			}
		}
	}

	// Default to allowing sharing if no explicit setting found
	return true, nil
}
