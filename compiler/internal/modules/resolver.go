package modules

import (
	"ferret/colors"
	"ferret/config"
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
	// Strip the "modules/" prefix for directory-based imports
	if after, ok := strings.CutPrefix(importPath, "modules/"); ok {
		importPath = after
	}

	modulePath = filepath.Join(modulePath, importPath+".fer")

	if fs.IsValidFile(modulePath) {
		return modulePath, nil
	}

	return "", fmt.Errorf("built-in module not found: %s", importPath)
}

// ResolveLocalModule resolves local project modules
func ResolveLocalModule(importPath, projectDirName string, projectRoot string) (string, error) {
	if projectDirName == "" {
		return "", fmt.Errorf("project directory name not defined")
	}

	importRoot := fs.FirstPart(importPath)
	if importRoot != projectDirName {
		return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
	}

	// Remove the project directory name from the import path and resolve relative to project root
	// e.g., "app/maths/math" becomes "maths/math"
	relativePath := strings.TrimPrefix(importPath, projectDirName+"/")
	resolvedPath := filepath.Join(projectRoot, relativePath+".fer")

	if fs.IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
}

// ResolveLocalProjectModule resolves modules from external local projects (like Go's replace directive)
func ResolveLocalProjectModule(importPath string, localsConfig map[string]string) (string, error) {
	if localsConfig == nil {
		return "", fmt.Errorf("no local projects configured")
	}

	importRoot := fs.FirstPart(importPath)
	localProjectPath, exists := localsConfig[importRoot]
	if !exists {
		return "", fmt.Errorf("local project `%s` not found in locals configuration", importRoot)
	}

	// Convert relative path to absolute if needed
	if !filepath.IsAbs(localProjectPath) {
		// Make it relative to the current working directory or project root
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
		}
		localProjectPath = filepath.Join(cwd, localProjectPath)
	}

	// Check if the local project path exists
	if stat, err := os.Stat(localProjectPath); err != nil || !stat.IsDir() {
		return "", fmt.Errorf("local project path `%s` does not exist or is not a directory", localProjectPath)
	}

	// Remove the project root from the import path and resolve relative to the local project path
	// e.g., "app2/file2/types" becomes "file2/types"
	relativePath := strings.TrimPrefix(importPath, importRoot+"/")
	resolvedPath := filepath.Join(localProjectPath, relativePath+".fer")

	if fs.IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in local project `%s`", relativePath, localProjectPath)
}

// ResolveNeighbourProjectModule resolves modules from external neighbouring projects (like Go's replace directive)
func ResolveNeighbourProjectModule(importPath string, neighbourConfig map[string]string) (string, error) {
	if neighbourConfig == nil {
		return "", fmt.Errorf("no neighbouring projects configured")
	}

	importRoot := fs.FirstPart(importPath)
	neighbourProjectPath, exists := neighbourConfig[importRoot]
	if !exists {
		return "", fmt.Errorf("neighbouring project `%s` not found in neighbour configuration", importRoot)
	}

	// Convert relative path to absolute if needed
	if !filepath.IsAbs(neighbourProjectPath) {
		// Make it relative to the current working directory or project root
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
		}
		neighbourProjectPath = filepath.Join(cwd, neighbourProjectPath)
	}

	// Check if the neighbouring project path exists
	if stat, err := os.Stat(neighbourProjectPath); err != nil || !stat.IsDir() {
		return "", fmt.Errorf("neighbouring project path `%s` does not exist or is not a directory", neighbourProjectPath)
	}

	// Remove the project root from the import path and resolve relative to the neighbouring project path
	// e.g., "app2/file2/types" becomes "file2/types"
	relativePath := strings.TrimPrefix(importPath, importRoot+"/")
	resolvedPath := filepath.Join(neighbourProjectPath, relativePath+".fer")

	if fs.IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in neighbouring project `%s`", relativePath, neighbourProjectPath)
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

	// For directory-based imports, we need to strip the remote module's directory name prefix
	// Extract the repository name from the cached path to determine the expected prefix
	remoteModulePrefix := getRemoteModuleDirName(remoteModuleRoot)

	actualImportPath := importPath
	if remoteModulePrefix != "" && strings.HasPrefix(importPath, remoteModulePrefix+"/") {
		// Strip the remote module prefix for internal imports
		actualImportPath = strings.TrimPrefix(importPath, remoteModulePrefix+"/")
	}

	// Resolve the import path relative to the remote module root
	resolvedPath := filepath.Join(remoteModuleRoot, actualImportPath+EXT)

	if fs.IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in remote module at %s", importPath, resolvedPath)
}

// getRemoteModuleDirName extracts the repository directory name from a remote module root path
// Example: github.com/itsfuad/ferret-mod@v1.0.1 -> ferret-mod
func getRemoteModuleDirName(remoteModuleRoot string) string {
	// Get the last part of the path which should be like "ferret-mod@v1.0.1"
	dirName := filepath.Base(remoteModuleRoot)

	// Strip version suffix if present
	if strings.Contains(dirName, "@") {
		dirName = strings.Split(dirName, "@")[0]
	}

	return dirName
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

	version, err := checkVersion(deps, repo)
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
			return "", fmt.Errorf("no module %q found in %q", importPath, repo)
		}
	}

	// ✅ SECURITY CHECK: Check if the target remote module allows sharing at project level
	canShare, err := CheckRemoteModuleShareSetting(moduleFullPath)
	if err != nil {
		return "", fmt.Errorf("failed to check share settings for module %s: %w", repo, err)
	}
	if !canShare {
		return "", fmt.Errorf("module %q has disabled sharing (share = false). Cannot import this module", repo)
	}

	return moduleFullPath, nil
}

func checkVersion(deps map[string]FerRetDependency, repo string) (string, error) {
	var version string
	dep, ok := deps[repo]
	if !ok || dep.Version == "" {
		// Module not found in fer.ret - this is now a strict requirement
		return "", fmt.Errorf("module %s is not declared in fer.ret dependencies.\n%s %s %s", repo, colors.YELLOW.Sprintf("Add it to fer.ret or run"), colors.BLUE.Sprintf("ferret get %s", repo), colors.YELLOW.Sprintf("to install and declare the dependency"))
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
		if after, ok := strings.CutPrefix(version, prefix); ok {
			version = after
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
			// For directory-based imports, strip the remote module directory name prefix
			remoteModuleRoot, err := findRemoteModuleRoot(currentFilePath, remoteCachePath)
			if err == nil {
				remoteModuleDirName := getRemoteModuleDirName(remoteModuleRoot)
				if remoteModuleDirName != "" && strings.HasPrefix(importPath, remoteModuleDirName+"/") {
					// Strip the remote module prefix for internal imports
					actualImportPath := strings.TrimPrefix(importPath, remoteModuleDirName+"/")
					return remotePrefix + "/" + actualImportPath
				}
			}

			// Fallback to original logic for non-directory-based imports
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
	configData, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}
	// update the dependency
	configData.Dependencies.Modules[repoName] = version
	//write back
	configData.Save()
	return nil
}

// RemoveFerRetDependency removes a dependency from the fer.ret file
func RemoveFerRetDependency(projectRoot, repoName string) error {
	configData, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}
	// remove the dependency
	delete(configData.Dependencies.Modules, repoName)
	//write back
	configData.Save()
	return nil
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
