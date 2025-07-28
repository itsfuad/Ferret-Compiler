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
// by reading its fer.ret configuration file at the project level
func CheckRemoteModuleShareSetting(moduleFilePath string) (bool, error) {
	// For project-level checking, we need to find the fer.ret file that applies to this specific module
	// We'll walk up from the module file to find the nearest fer.ret file

	configPath, err := findProjectConfigForModule(moduleFilePath)
	if err != nil {
		return false, err
	}

	// If no fer.ret file found, assume sharing is allowed (default behavior)
	if configPath == "" {
		return true, nil
	}

	// Parse the fer.ret file in the remote module project
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

// findProjectConfigForModule finds the fer.ret file that applies to a specific module file
// by walking up the directory tree from the module location
func findProjectConfigForModule(moduleFilePath string) (string, error) {
	// For import "github.com/user/repo/data/bigint", moduleFilePath might be:
	// ".../cache/github.com/user/repo@v1/data/bigint.fer"
	// We need to find the fer.ret that applies to this module

	// Get the directory containing the module file
	currentDir := filepath.Dir(moduleFilePath)

	// Walk up the directory tree to find fer.ret
	for {
		configPath := filepath.Join(currentDir, "fer.ret")
		if _, err := os.Stat(configPath); err == nil {
			// Found fer.ret file
			return configPath, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)

		// Stop if we can't go up further or reached root
		if parentDir == currentDir {
			break
		}

		// Stop if we've left the cache directory structure
		if !strings.Contains(currentDir, "github.com") {
			break
		}

		currentDir = parentDir
	}

	// No fer.ret found
	return "", nil
}
