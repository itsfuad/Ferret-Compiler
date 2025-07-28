package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"compiler/colors"
	"compiler/internal/config"
	"compiler/internal/ctx"
	"compiler/internal/modules"
	"compiler/toml"
)

const EXT = ".fer"

const INVALID_GITHUB_PATH_MSG = "invalid GitHub repository path: %s"

// FerRetDependency represents a dependency entry in fer.ret
type FerRetDependency struct {
	Version string
	Comment string // Optional comment like "used by X"
}

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
		return resolveRemoteModule(importPath, ctxx)
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
	// Check if this is a remote import (starts with github.com, etc.)
	if strings.HasPrefix(importPath, "github.com/") {
		// This is a remote import, handle it normally
		return resolveRemoteModule(importPath, ctxx)
	}

	// This is a local import within the remote module
	// Find the root directory of the remote module
	remoteModuleRoot, err := findRemoteModuleRoot(currentFileFullPath, ctxx)
	if err != nil {
		return "", err
	}

	// Resolve the import path relative to the remote module root
	resolvedPath := filepath.Join(remoteModuleRoot, importPath+EXT)

	if modules.IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in remote module at %s", importPath, resolvedPath)
}

// findRemoteModuleRoot finds the root directory of the remote module containing the given file
func findRemoteModuleRoot(filePath string, ctxx *ctx.CompilerContext) (string, error) {
	// Get absolute paths for comparison
	absCachePath, err := filepath.Abs(ctxx.RemoteCachePath)
	if err != nil {
		return "", err
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// Get relative path from cache root
	relPath, err := filepath.Rel(absCachePath, absFilePath)
	if err != nil {
		return "", err
	}

	// Convert to forward slashes for consistent processing
	relPath = filepath.ToSlash(relPath)

	// Split the path: github.com/user/repo@version/sub/path/file.fer
	// We want to get: github.com/user/repo@version
	parts := strings.Split(relPath, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid remote module path structure: %s", relPath)
	}

	// Take the first 3 parts: github.com/user/repo@version
	moduleRelPath := strings.Join(parts[:3], "/")
	moduleRoot := filepath.Join(absCachePath, moduleRelPath)

	return filepath.ToSlash(moduleRoot), nil
}

// resolveRemoteModule resolves remote module imports
func resolveRemoteModule(importPath string, ctxx *ctx.CompilerContext) (string, error) {
	// Parse the remote import to get repo information
	_, _, repoName, err := ParseRemoteImport(importPath)
	if err != nil {
		return "", fmt.Errorf("invalid remote import path: %w", err)
	}

	// Check if this repo is in fer.ret dependencies
	dependencies, err := ReadFerRetDependencies(ctxx.ProjectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read dependencies from project root '%s': %w", ctxx.ProjectRoot, err)
	}

	// Check if the repo is listed in dependencies using full repo path
	fullRepoPath := "github.com/" + repoName
	dependency, exists := dependencies[fullRepoPath]
	if !exists {
		return "", fmt.Errorf("module %s is not installed. run ferret get %s to install", fullRepoPath, fullRepoPath)
	}

	// Check if module is cached with the installed version
	if !IsModuleCached(ctxx.RemoteCachePath, repoName, dependency.Version) {
		return "", fmt.Errorf("module %s is listed in fer.ret but not found in cache. run ferret get %s to reinstall", fullRepoPath, fullRepoPath)
	}

	// Build the module file path within the cache
	// The import path might include subdirectories after the repo name
	// Example: "github.com/user/repo/folder1/folder2/module"
	// -> repo is "user/repo", module path is "folder1/folder2/module"

	// Remove the github.com/user/repo prefix to get the module path within the repo
	// We already have fullRepoPath from above: github.com/user/repo
	moduleDir := filepath.Join(ctxx.RemoteCachePath, "github.com", repoName+"@"+dependency.Version)

	var modulePath string
	if len(importPath) > len(fullRepoPath) {
		modulePath = importPath[len(fullRepoPath)+1:] // +1 to skip the trailing slash
	}

	var moduleFullPath string
	if modulePath == "" {
		// Import is just the repo root, no specific module path
		moduleFullPath = filepath.Join(moduleDir, "fer.ret")
		if _, err := os.Stat(moduleFullPath); os.IsNotExist(err) {
			return "", fmt.Errorf("module root file not found at %s", moduleFullPath)
		}
	} else {
		// Build full path to the specific module
		moduleFullPath = filepath.Join(moduleDir, modulePath+".fer")
		if _, err := os.Stat(moduleFullPath); os.IsNotExist(err) {
			return "", fmt.Errorf("module file not found at %s", moduleFullPath)
		}
	}

	// ✅ SECURITY CHECK: Check if the target remote module allows sharing at project level
	canShare, err := modules.CheckRemoteModuleShareSetting(moduleFullPath)
	if err != nil {
		return "", fmt.Errorf("failed to check share settings for module %s: %w", fullRepoPath, err)
	}
	if !canShare {
		return "", fmt.Errorf("module %s has disabled sharing (share = false). Cannot import this module", fullRepoPath)
	}

	return moduleFullPath, nil
}

// CheckCanImportRemoteModules validates if remote imports are allowed for the current project
func CheckCanImportRemoteModules(ctx *ctx.CompilerContext, importPath string) error {
	// Only check for remote imports (starting with github.com/)
	if !strings.HasPrefix(importPath, "github.com/") {
		return nil // Not a remote import, allow it
	}

	// Load project configuration to check remote settings
	projectConfig, err := config.LoadProjectConfig(ctx.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to load project configuration: %w", err)
	}

	// ✅ SECURITY CHECK: Check if remote imports are enabled
	if !projectConfig.Remote.Enabled {
		return fmt.Errorf("remote module imports are disabled in this project. To enable, set 'enabled = true' in the [remote] section of fer.ret")
	}

	return nil
}

// ResolveImportPath resolves import paths based on the current file context
// This mirrors the logic in the collector phase
func ResolveImportPath(importPath, currentFilePath string, ctx *ctx.CompilerContext) string {
	// If this is already a full remote path, return as-is
	if strings.HasPrefix(importPath, "github.com/") {
		return importPath
	}

	// Check if the current file is in a remote module cache
	if IsFileInRemoteCache(currentFilePath, ctx) {
		// This is a local import within a remote module
		// Convert it to the full GitHub path
		remotePrefix := GetRemoteModulePrefix(currentFilePath, ctx)
		if remotePrefix != "" {
			return remotePrefix + "/" + importPath
		}
	}

	// For all other cases (local project imports, builtin modules), return as-is
	return importPath
}

// ParseRemoteImport extracts repo and version information from a remote import path
// Example: "github.com/user/repo@v1.0.0" -> ("github.com/user/repo", "v1.0.0", "user/repo")
func ParseRemoteImport(importPath string) (repoPath, version, repoName string, err error) {
	if !strings.HasPrefix(importPath, "github.com/") {
		return "", "", "", fmt.Errorf("only github.com repositories are supported")
	}

	// Check for version specification
	if strings.Contains(importPath, "@") {
		parts := strings.Split(importPath, "@")
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("invalid version specification in import path: %s", importPath)
		}
		repoPath = parts[0]
		version = parts[1]
	} else {
		repoPath = importPath
		version = "latest"
	}

	// Extract repo name (user/repo) from github.com/user/repo
	pathParts := strings.Split(repoPath, "/")
	if len(pathParts) < 3 {
		return "", "", "", fmt.Errorf("invalid GitHub repository path: %s", repoPath)
	}

	repoName = strings.Join(pathParts[1:3], "/") // "user/repo"

	return repoPath, version, repoName, nil
}

// ReadFerRetDependencies reads the dependencies section from fer.ret file
func ReadFerRetDependencies(projectRoot string) (map[string]FerRetDependency, error) {
	ferRetPath := filepath.Join(projectRoot, FerretConfigFile)

	// Check if file exists first
	if _, err := os.Stat(ferRetPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("fer.ret file not found at %s", ferRetPath)
	}

	// Use the existing TOML parser
	data, err := toml.ParseTOMLFile(ferRetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fer.ret at %s: %w", ferRetPath, err)
	}

	dependencies := make(map[string]FerRetDependency)

	// Get dependencies section - it's okay if it doesn't exist
	if depsSection, exists := data["dependencies"]; exists {
		for key, value := range depsSection {
			// Skip comments (lines starting with #)
			if strings.HasPrefix(key, "#") {
				continue
			}

			var version string

			// Value should be a string version
			if versionStr, ok := value.(string); ok {
				version = versionStr
			} else {
				// Convert other types to string
				version = fmt.Sprintf("%v", value)
			}

			dependencies[key] = FerRetDependency{
				Version: version,
				Comment: "", // Comments are handled separately now
			}
		}
	}
	// If dependencies section doesn't exist, return empty map (valid for new projects)

	return dependencies, nil
}

// WriteFerRetDependency adds or updates a dependency in the fer.ret file
func WriteFerRetDependency(projectRoot, repoName, version, comment string) error {
	ferRetPath := filepath.Join(projectRoot, FerretConfigFile)

	// Read existing content
	data, err := toml.ParseTOMLFile(ferRetPath)
	if err != nil {
		return fmt.Errorf("failed to parse fer.ret: %w", err)
	}

	// Ensure dependencies section exists in the original data
	if _, exists := data["dependencies"]; !exists {
		data["dependencies"] = make(toml.TOMLTable)
	}

	// Add/update the dependency
	data["dependencies"][repoName] = version

	// Prepare inline comments if provided
	var inlineComments map[string]map[string]string
	if comment != "" {
		inlineComments = map[string]map[string]string{
			"dependencies": {
				repoName: comment,
			},
		}
	}

	// Write back to file using the TOML writer
	return toml.WriteTOMLFile(ferRetPath, data, inlineComments)
}

// RemoveFerRetDependency removes a dependency from the fer.ret file
func RemoveFerRetDependency(projectRoot, repoName string) error {
	ferRetPath := filepath.Join(projectRoot, FerretConfigFile)

	// Read existing content
	data, err := toml.ParseTOMLFile(ferRetPath)
	if err != nil {
		return fmt.Errorf("failed to parse fer.ret: %w", err)
	}

	// Remove from dependencies section
	if depsSection, exists := data["dependencies"]; exists {
		delete(depsSection, repoName)
	}

	// Write back to file
	return toml.WriteTOMLFile(ferRetPath, data, nil)
}
