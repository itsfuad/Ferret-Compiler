package registry

import (
	"fmt"
	"path/filepath"
	"strings"

	"compiler/colors"
	"compiler/internal/config"
	"compiler/internal/ctx"
	"compiler/internal/modules"
)

const EXT = ".fer"

const INVALID_GITHUB_PATH_MSG = "invalid GitHub repository path: %s"

// ResolveModuleLocation resolves the full path of a module based on its import path
// It returns the physical location of the module file on disk
// or an error if the module cannot be found.
func ResolveModuleLocation(importPath, currentFileFullPath string, ctxx *ctx.CompilerContext) (string, error) {
	// Validate import path
	if importPath == "" {
		return "", fmt.Errorf("import path cannot be empty")
	}

	// Get project name for local module resolution
	projectName := ""
	if ctxx.ProjectConfig != nil {
		projectName = ctxx.ProjectConfig.Name
	}

	// Determine module type
	moduleType := modules.GetModuleType(importPath, projectName)

	// Handle special case: if current file is in remote cache, imports should be in remote context
	if isFileCached(currentFileFullPath, ctxx) {
		return resolveModuleInRemoteContext(importPath, currentFileFullPath, ctxx)
	}

	// Route to appropriate resolver based on module type
	switch moduleType {
	case modules.REMOTE:
		return "", fmt.Errorf("remote is not supported yet")
	case modules.BUILTIN:
		return resolveBuiltinModule(importPath, ctxx)
	case modules.LOCAL:
		return resolveLocalModule(importPath, projectName, ctxx)
	default:
		return "", fmt.Errorf("unknown module type for import: %s", importPath)
	}
}

// resolveBuiltinModule resolves built-in system modules
func resolveBuiltinModule(importPath string, ctxx *ctx.CompilerContext) (string, error) {
	modulePath := filepath.Join(ctxx.ModulesPath, importPath+EXT)
	colors.AQUA.Printf("Searching for built-in module: %s -> %s\n", importPath, modulePath)

	if modules.IsValidFile(modulePath) {
		return modulePath, nil
	}

	return "", fmt.Errorf("built-in module not found: %s", importPath)
}

// resolveLocalModule resolves local project modules
func resolveLocalModule(importPath, projectName string, ctxx *ctx.CompilerContext) (string, error) {
	if projectName == "" {
		return "", fmt.Errorf("project name not defined in configuration")
	}

	importRoot := modules.FirstPart(importPath)
	if importRoot != projectName {
		return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
	}

	// Remove the project name from the import path and resolve relative to project root
	// e.g., "myapp/maths/math" becomes "maths/math"
	relativePath := strings.TrimPrefix(importPath, projectName+"/")
	resolvedPath := filepath.Join(ctxx.ProjectRoot, relativePath+EXT)

	if modules.IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
}

// isFileCached checks if the given file path is inside the remote cache directory
func isFileCached(filePath string, ctxx *ctx.CompilerContext) bool {
	// If file path is empty, it's not in remote cache
	if filePath == "" {
		return false
	}

	// Normalize paths for comparison
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	absCachePath, err := filepath.Abs(ctxx.RemoteCachePath)
	if err != nil {
		return false
	}

	// Check if the file is inside the remote cache directory
	return strings.HasPrefix(absFilePath, absCachePath)
}

// resolveModuleInRemoteContext resolves a module import within a remote module's context
func resolveModuleInRemoteContext(importPath, currentFileFullPath string, ctxx *ctx.CompilerContext) (string, error) {
	// Find the directory containing the fer.ret file for the current file's module
	remoteModuleConfigDir, err := findRemoteModuleConfigDir(currentFileFullPath, ctxx)
	if err != nil {
		return "", err
	}

	// Parse the remote module's configuration (LoadProjectConfig expects directory path)
	remoteConfig, err := config.LoadProjectConfig(remoteModuleConfigDir)
	if err != nil {
		return "", fmt.Errorf("failed to load remote module config: %w", err)
	}

	// Get the first part of the import path (the project name)
	importRoot := modules.FirstPart(importPath)

	// Check if this import matches the remote module's project name
	if importRoot == remoteConfig.Name {
		// Remove the project name from the import path and resolve relative to remote module config dir
		relativePath := strings.TrimPrefix(importPath, remoteConfig.Name+"/")
		resolvedPath := filepath.Join(remoteModuleConfigDir, relativePath+EXT)

		if modules.IsValidFile(resolvedPath) {
			return resolvedPath, nil
		}
		return "", fmt.Errorf("module `%s` does not exist in remote module", importPath)
	}

	return "", fmt.Errorf("module `%s` does not exist in remote module", importPath)
}

// findRemoteModuleConfigDir finds the directory containing the fer.ret for the module containing the given file
func findRemoteModuleConfigDir(filePath string, ctxx *ctx.CompilerContext) (string, error) {
	// Start from the directory containing the file and walk up looking for fer.ret
	currentDir := filepath.Dir(filePath)

	// Get the cache path to ensure we don't walk outside the cache
	absCachePath, err := filepath.Abs(ctxx.RemoteCachePath)
	if err != nil {
		return "", err
	}

	for {
		// Check if fer.ret exists in current directory
		configPath := filepath.Join(currentDir, "fer.ret")
		if modules.IsValidFile(configPath) {
			return currentDir, nil
		}

		// Move up one level
		parentDir := filepath.Dir(currentDir)

		// Stop if we've reached the cache root or can't go up further
		if parentDir == currentDir || !strings.HasPrefix(currentDir, absCachePath) {
			break
		}

		currentDir = parentDir
	}

	return "", fmt.Errorf("no fer.ret found in remote module hierarchy for: %s", filePath)
}