package modules

import (
	"ferret/colors"
	"ferret/internal/config"
	"ferret/internal/utils/fs"
	"ferret/toml"
	"os"

	"fmt"
	"path/filepath"
	"strings"
)

const EXT = ".fer"
const CONFIG_FILE = "fer.ret"

// FerRetDependency represents a dependency entry in fer.ret
type FerRetDependency struct {
	Version string
	Comment string // Optional comment like "used by X"
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
	// Properly parse the TOML file and use the result
	configData, err := toml.ParseTOMLFile(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to parse TOML file: %w", err)
	}
	if remoteSection, exists := configData["remote"]; exists {
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
		configPath := filepath.Join(currentDir, CONFIG_FILE)
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

// ResolveBuiltinModule resolves built-in system modules
func ResolveBuiltinModule(importPath string, modulePath string) (string, error) {

	modulePath = filepath.Join(modulePath, importPath+".fer")
	colors.AQUA.Printf("Searching for built-in module: %s -> %s\n", importPath, modulePath)

	if fs.IsValidFile(modulePath) {
		return modulePath, nil
	}

	return "", fmt.Errorf("built-in module not found: %s", importPath)
}

// ResolveLocalModule resolves local project modules
func ResolveLocalModule(importPath, projectName string, projectRoot string) (string, error) {
	if projectName == "" {
		return "", fmt.Errorf("project name not defined in configuration")
	}

	importRoot := fs.FirstPart(importPath)
	if importRoot != projectName {
		return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
	}

	// Remove the project name from the import path and resolve relative to project root
	// e.g., "myapp/maths/math" becomes "maths/math"
	relativePath := strings.TrimPrefix(importPath, projectName+"/")
	resolvedPath := filepath.Join(projectRoot, relativePath+".fer")

	if fs.IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
}

// IsFilepathInCache checks if the given file path is inside the remote cache directory
func IsFilepathInCache(filePath string, remoteCachePath string) bool {
	// If file path is empty, it's not in remote cache
	if filePath == "" {
		return false
	}

	// Normalize paths for comparison
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	absCachePath, err := filepath.Abs(remoteCachePath)
	if err != nil {
		return false
	}

	// Check if the file is inside the remote cache directory
	return strings.HasPrefix(absFilePath, absCachePath)
}

// ResolveModuleInRemoteContext resolves a module import within a remote module's context
func ResolveModuleInRemoteContext(importPath, currentFileFullPath string, projectRoot, remoteCachePath string) (string, error) {
	// Check if this is a remote import (starts with github.com, etc.)
	if strings.HasPrefix(importPath, REMOTE_HOST) {
		// This is a remote import, handle it normally
		return ResolveRemoteModule(importPath, projectRoot, remoteCachePath, currentFileFullPath)
	}

	// This is a local import within the remote module
	// Find the root directory of the remote module
	remoteModuleRoot, err := findRemoteModuleRoot(currentFileFullPath, remoteCachePath)
	if err != nil {
		return "", err
	}

	// Resolve the import path relative to the remote module root
	resolvedPath := filepath.Join(remoteModuleRoot, importPath+EXT)

	if fs.IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in remote module at %s", importPath, resolvedPath)
}

// findRemoteModuleRoot finds the root directory of the remote module containing the given file
func findRemoteModuleRoot(filePath string, remoteCachePath string) (string, error) {
	// Get absolute paths for comparison
	absCachePath, err := filepath.Abs(remoteCachePath)
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

// ResolveRemoteModule resolves remote module imports
func ResolveRemoteModule(importPath string, projectRoot, remoteCachePath string, importingFile string) (string, error) {
	// Parse the remote import to get repo information
	_, _, repoName, err := SplitRemotePath(importPath)
	if err != nil {
		return "", fmt.Errorf("invalid remote import path: %w", err)
	}

	// Walk up from the importing file to find the nearest fer.ret
	ferretPath := FindNearestFerRet(filepath.Dir(importingFile), projectRoot)

	// Read dependencies from the found fer.ret
	deps, err := readDependenciesFromFerRetFile(ferretPath)
	if err != nil {
		return "", fmt.Errorf("failed to read fer.ret: %w", err)
	}

	// Get the version for the imported repo
	repo := REMOTE_HOST + repoName

	version, err := checkVersion(deps, repo, projectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to check version for module %s: %w", repo, err)
	}

	// Load lockfile to check if this repo@version is installed
	lockfile, err := LoadLockfile(projectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to load lockfile: %w", err)
	}
	key := repo + "@" + version
	entry, exists := lockfile.Dependencies[key]
	if !exists {
		return "", fmt.Errorf("module %s@%s is listed in fer.ret but not found in cache\n%s %s %s", repo, version, colors.YELLOW.Sprintf("run"), colors.BLUE.Sprintf("ferret get"), colors.YELLOW.Sprintf("to auto install"))
	}

	// Check if module is cached with the installed version
	if !IsModuleCached(remoteCachePath, repoName, entry.Version) {
		return "", fmt.Errorf("module %s@%s is listed in lockfile but not found in cache\n%s %s", repo, version, colors.YELLOW.Sprintf("run"), colors.BLUE.Sprintf("ferret get to reinstall all dependencies"))
	}

	// Build the module file path within the cache
	moduleDir := filepath.Join(remoteCachePath, "github.com", repoName+"@"+version)

	// Remove the github.com/user/repo prefix to get the module path within the repo
	fullRepoPath := repo
	var modulePath string
	if len(importPath) > len(fullRepoPath) {
		modulePath = importPath[len(fullRepoPath)+1:] // +1 to skip the trailing slash
	}

	var moduleFullPath string
	if modulePath == "" {
		// Import is just the repo root, no specific module path
		moduleFullPath = filepath.Join(moduleDir, CONFIG_FILE)
		if _, err := os.Stat(moduleFullPath); os.IsNotExist(err) {
			return "", fmt.Errorf("no project found for module %s", importPath)
		}
	} else {
		// Build full path to the specific module
		moduleFullPath = filepath.Join(moduleDir, modulePath+".fer")
		if _, err := os.Stat(moduleFullPath); os.IsNotExist(err) {
			return "", fmt.Errorf("module %s not found in %s", importPath, repo)
		}
	}

	// ✅ SECURITY CHECK: Check if the target remote module allows sharing at project level
	canShare, err := CheckRemoteModuleShareSetting(moduleFullPath)
	if err != nil {
		return "", fmt.Errorf("failed to check share settings for module %s: %w", repo, err)
	}
	if !canShare {
		return "", fmt.Errorf("module %s has disabled sharing (share = false). Cannot import this module", repo)
	}

	return moduleFullPath, nil
}

func checkVersion(deps map[string]FerRetDependency, repo string, projectRoot string) (string, error) {
	var version string
	dep, ok := deps[repo]
	if !ok || dep.Version == "" {
		// Module not found in fer.ret - check lockfile for any installed version
		lockfile, err := LoadLockfile(projectRoot)
		if err != nil {
			return "", fmt.Errorf("failed to load lockfile: %w", err)
		}

		// Search for any installed version of this repo
		var foundVersions []string
		for key, entry := range lockfile.Dependencies {
			if strings.HasPrefix(key, repo+"@") {
				foundVersions = append(foundVersions, entry.Version)
			}
		}

		if len(foundVersions) == 0 {
			return "", fmt.Errorf("module %s@%s is not installed\n%s %s %s", repo, version, colors.YELLOW.Sprintf("run"), colors.BLUE.Sprintf("ferret get %s", repo), colors.YELLOW.Sprintf("to install"))
		}

		if len(foundVersions) == 1 {
			// Exactly one version found - use it
			version = foundVersions[0]
		} else {
			// Multiple versions found - ask user to specify
			return "", fmt.Errorf("multiple versions of %s are installed: %v. Please specify the version in your fer.ret file", repo, foundVersions)
		}
	} else {
		version = dep.Version
	}

	return version, nil
}

// Helper to read dependencies from a specific fer.ret file
func readDependenciesFromFerRetFile(ferretPath string) (map[string]FerRetDependency, error) {
	// Use the existing TOML parser
	data, err := toml.ParseTOMLFile(ferretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fer.ret at %s: %w", ferretPath, err)
	}
	dependencies := make(map[string]FerRetDependency)
	if depsSection, exists := data["dependencies"]; exists {
		for key, value := range depsSection {
			if strings.HasPrefix(key, "#") {
				continue
			}
			var version string
			if versionStr, ok := value.(string); ok {
				version = versionStr
			} else {
				version = fmt.Sprintf("%v", value)
			}

			// Strip common version prefixes for lockfile compatibility
			version = stripVersionPrefix(version)

			dependencies[key] = FerRetDependency{
				Version: version,
				Comment: "",
			}
		}
	}
	return dependencies, nil
}

// stripVersionPrefix removes common version prefixes like ^, ~, >=, etc.
func stripVersionPrefix(version string) string {
	// Remove common prefixes
	prefixes := []string{"^", "~", ">=", "<=", ">", "<", "="}
	for _, prefix := range prefixes {
		if strings.HasPrefix(version, prefix) {
			version = strings.TrimPrefix(version, prefix)
			break
		}
	}
	return version
}

// CheckCanImportRemoteModules validates if remote imports are allowed for the current project
func CheckCanImportRemoteModules(projectRoot string, importPath string) error {
	// Only check for remote imports (starting with github.com/)
	if !strings.HasPrefix(importPath, REMOTE_HOST) {
		return nil // Not a remote import, allow it
	}

	// Load project configuration to check remote settings
	projectConfig, err := config.LoadProjectConfig(projectRoot)
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
func ResolveImportPath(importPath, currentFilePath string, remoteCachePath string) string {
	// If this is already a full remote path, return as-is
	if strings.HasPrefix(importPath, REMOTE_HOST) {
		return importPath
	}

	// Check if the current file is in a remote module cache
	if IsFileInRemoteCache(currentFilePath, remoteCachePath) {
		// This is a local import within a remote module
		// Convert it to the full GitHub path
		remotePrefix := GetRemoteModulePrefix(currentFilePath, remoteCachePath)
		if remotePrefix != "" {
			return remotePrefix + "/" + importPath
		}
	}

	// For all other cases (local project imports, builtin modules), return as-is
	return importPath
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
func WriteFerRetDependency(projectRoot, repoName, version, comment string, isCached bool) error {
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

	if isCached {
		colors.ORANGE.Printf("🔄️Reusing cached module: %s@%s\n", repoName, version)
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

// FindNearestFerRet walks up from a starting directory to find the nearest fer.ret file.
func FindNearestFerRet(startingDir, projectRoot string) string {
	dir := startingDir
	for {
		ferretPath := filepath.Join(dir, CONFIG_FILE)
		if _, err := os.Stat(ferretPath); err == nil {
			return ferretPath
		}
		parent := filepath.Dir(dir)
		if parent == dir || dir == projectRoot {
			break
		}
		dir = parent
	}
	// Fallback to project root
	return filepath.Join(projectRoot, CONFIG_FILE)
}
