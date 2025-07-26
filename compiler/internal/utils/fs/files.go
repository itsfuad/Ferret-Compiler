package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"compiler/colors"
	"compiler/internal/config"
	"compiler/internal/ctx"
	"compiler/internal/registry"
)

const EXT = ".fer"
const REMOTE_HOST = "github.com/"

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

// DetermineModuleType categorizes an import path
func DetermineModuleType(importPath string, projectName string) ctx.ModuleType {
	importRoot := FirstPart(importPath)

	if IsRemote(importPath) {
		return ctx.REMOTE
	}

	if IsBuiltinModule(importRoot) {
		return ctx.BUILTIN
	}

	if importRoot == projectName {
		return ctx.LOCAL
	}

	// Default to local for unrecognized paths
	return ctx.LOCAL
}

func IsRemote(importPath string) bool {
	return strings.HasPrefix(importPath, REMOTE_HOST)
}

// Check if file exists and is a regular file
func IsValidFile(filename string) bool {
	fileInfo, err := os.Stat(filepath.FromSlash(filename))
	return err == nil && fileInfo.Mode().IsRegular()
}

// GitHubPathToRawURL converts a GitHub import path to a raw.githubusercontent.com URL.
// Example: "github.com/user/repo/path/file" → "https://raw.githubusercontent.com/user/repo/main/path/file"
func GitHubPathToRawURL(importPath, defaultBranch string) (string, string) {
	if !strings.HasPrefix(importPath, REMOTE_HOST) {
		return "", ""
	}
	parts := strings.SplitN(importPath, "/", 4)
	if len(parts) < 4 {
		return "", ""
	}
	user := parts[1]
	repo := parts[2]
	subpath := parts[3]

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s.fer",
		user, repo, defaultBranch, subpath,
	)

	return url, subpath
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

func ResolveModule(importPath, currentFileFullPath string, ctxx *ctx.CompilerContext) (string, error) {
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
	moduleType := DetermineModuleType(importPath, projectName)

	// Handle special case: if current file is in remote cache, local imports should be remote
	if moduleType == ctx.LOCAL && isFileInRemoteCache(currentFileFullPath, ctxx) {
		return resolveModuleInRemoteContext(importPath, currentFileFullPath, ctxx)
	}

	// Route to appropriate resolver based on module type
	switch moduleType {
	case ctx.REMOTE:
		return resolveRemoteModuleNew(importPath, ctxx)
	case ctx.BUILTIN:
		return resolveBuiltinModule(importPath, ctxx)
	case ctx.LOCAL:
		return resolveLocalModule(importPath, projectName, ctxx)
	default:
		return "", fmt.Errorf("unknown module type for import: %s", importPath)
	}
}

// resolveBuiltinModule resolves built-in system modules
func resolveBuiltinModule(importPath string, ctxx *ctx.CompilerContext) (string, error) {
	modulePath := filepath.Join(ctxx.ModulesPath, importPath+EXT)
	colors.AQUA.Printf("Searching for built-in module: %s -> %s\n", importPath, modulePath)

	if IsValidFile(modulePath) {
		return modulePath, nil
	}

	return "", fmt.Errorf("built-in module not found: %s", importPath)
}

// resolveLocalModule resolves local project modules
func resolveLocalModule(importPath, projectName string, ctxx *ctx.CompilerContext) (string, error) {
	if projectName == "" {
		return "", fmt.Errorf("project name not defined in configuration")
	}

	importRoot := FirstPart(importPath)
	if importRoot != projectName {
		return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
	}

	// Remove the project name from the import path and resolve relative to project root
	// e.g., "myapp/maths/math" becomes "maths/math"
	relativePath := strings.TrimPrefix(importPath, projectName+"/")
	resolvedPath := filepath.Join(ctxx.ProjectRoot, relativePath+EXT)

	if IsValidFile(resolvedPath) {
		return resolvedPath, nil
	}

	return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
}

// resolveRemoteModuleNew implements the new import system workflow:
// 1. Check fer.ret for dependency declaration
// 2. Check ferret.lock for version/dependency info
// 3. Check cache and auto-install if needed
func resolveRemoteModuleNew(importPath string, ctxx *ctx.CompilerContext) (string, error) {
	repoPath, requestedVersion, subPath := ctxx.ParseRemoteImport(importPath)

	if repoPath == "" {
		return "", fmt.Errorf("invalid remote import path: %s", importPath)
	}

	// Step 1: Check fer.ret for dependency declaration
	dependencies, err := registry.ParseFerRetDependencies(ctxx.ProjectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read fer.ret: %w", err)
	}

	declaredVersion, isDeclared := dependencies[repoPath]
	if !isDeclared {
		return "", fmt.Errorf("module '%s' is not declared in fer.ret\n\nPlease add it to the [dependencies] section:\n\n[dependencies]\n%s = \"v1.0.0\"\n\nThen run: ferret get %s", repoPath, repoPath, repoPath)
	}

	// Use declared version if no specific version was requested
	targetVersion := declaredVersion
	if requestedVersion != "" && requestedVersion != "latest" {
		// Validate that requested version matches declared version
		if requestedVersion != declaredVersion {
			colors.YELLOW.Printf("Warning: Requested version %s differs from declared version %s for %s, using declared version\n",
				requestedVersion, declaredVersion, repoPath)
		}
	}

	// Step 2: Check ferret.lock for dependency info
	lockFile, err := registry.LoadLockFile(ctxx.ProjectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read ferret.lock: %w", err)
	}

	// Step 3: Resolve version using flat dependency structure
	flatModuleName := repoPath + "@" + targetVersion

	// Check if this specific version is in cache
	if !ctxx.IsRemoteModuleCachedFlat(flatModuleName) {
		// Auto-install from fer.ret and lock file data
		colors.CYAN.Printf("Installing module %s@%s...\n", repoPath, targetVersion)
		err := autoInstallModule(repoPath, targetVersion, lockFile, ctxx)
		if err != nil {
			return "", fmt.Errorf("failed to auto-install module %s@%s: %w", repoPath, targetVersion, err)
		}
	}

	// Construct path to the cached module file using flat structure
	return resolveCachedModulePathFlat(flatModuleName, subPath, ctxx)
}

// autoInstallModule automatically installs a module based on fer.ret and lock file data
func autoInstallModule(repoPath, version string, lockFile *registry.LockFile, ctxx *ctx.CompilerContext) error {
	// This will trigger the download/installation process
	// We'll implement this to call the existing remote installation logic
	colors.GREEN.Printf("Auto-installing %s@%s from declared dependencies...\n", repoPath, version)

	// Call the existing download function with the flat structure support
	err := registry.DownloadRemoteModule(ctxx, repoPath, version)
	if err != nil {
		return fmt.Errorf("failed to download module %s@%s: %w", repoPath, version, err)
	}

	colors.GREEN.Printf("Successfully auto-installed %s@%s\n", repoPath, version)
	return nil
}

// resolveCachedModulePathFlat resolves module path using flat cache structure
func resolveCachedModulePathFlat(flatModuleName, subPath string, ctxx *ctx.CompilerContext) (string, error) {
	// Construct path using flat structure: .ferret/modules/github.com/user/repo@version/
	cachePath := filepath.Join(ctxx.RemoteCachePath, flatModuleName)

	var modulePath string
	if subPath != "" {
		modulePath = filepath.Join(cachePath, subPath+EXT)
	} else {
		// Look for common entry point names
		possibleFiles := []string{"index.fer", "main.fer", "mod.fer"}
		for _, fileName := range possibleFiles {
			candidatePath := filepath.Join(cachePath, fileName)
			if IsValidFile(candidatePath) {
				modulePath = candidatePath
				break
			}
		}

		if modulePath == "" {
			return "", fmt.Errorf("no entry point found in cached module: %s", flatModuleName)
		}
	}

	if !IsValidFile(modulePath) {
		return "", fmt.Errorf("module file not found in cache: %s", modulePath)
	}

	return modulePath, nil
}

// resolveVersionForCache resolves version like "latest" to actual cached version
func resolveVersionForCache(repoPath, version string, ctxx *ctx.CompilerContext) (string, error) {
	if version != "latest" {
		return version, nil
	}

	// For "latest", we need to find what version was actually cached
	// The cache structure is: .ferret/modules/github.com/user/repo@version
	// So we need to look in the directory containing all versions of this repo
	baseRepoPath := filepath.Join(ctxx.RemoteCachePath, repoPath)
	parentDir := filepath.Dir(baseRepoPath)
	repoName := filepath.Base(repoPath) // e.g., "ferret-mod"

	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return "", fmt.Errorf("failed to read cache directory: %w", err)
	}

	// Find directories that start with repoName@
	prefix := repoName + "@"
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			// Extract version from directory name
			actualVersion := strings.TrimPrefix(entry.Name(), prefix)
			return actualVersion, nil
		}
	}

	return "", fmt.Errorf("no cached version found for %s", repoPath)
}

// isFileInRemoteCache checks if the given file path is inside a remote module cache
func isFileInRemoteCache(filePath string, ctxx *ctx.CompilerContext) bool {
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
	importRoot := FirstPart(importPath)

	// Check if this import matches the remote module's project name
	if importRoot == remoteConfig.Name {
		// Remove the project name from the import path and resolve relative to remote module config dir
		relativePath := strings.TrimPrefix(importPath, remoteConfig.Name+"/")
		resolvedPath := filepath.Join(remoteModuleConfigDir, relativePath+EXT)

		if IsValidFile(resolvedPath) {
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
		if IsValidFile(configPath) {
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

// findRemoteModuleRoot finds the root directory of the remote module containing the given file
func findRemoteModuleRoot(filePath string, ctxx *ctx.CompilerContext) (string, error) {
	// Normalize the file path
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	absCachePath, err := filepath.Abs(ctxx.RemoteCachePath)
	if err != nil {
		return "", err
	}

	// Remove the cache path prefix to get the relative path within cache
	relPath, err := filepath.Rel(absCachePath, absFilePath)
	if err != nil {
		return "", err
	}

	// The structure is: github.com/user/repo@version/...
	// We need to find the repo@version directory
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid remote module path structure")
	}

	// Find the part that contains @ (the repo@version part)
	var repoVersionIndex int = -1
	for i, part := range parts {
		if strings.Contains(part, "@") {
			repoVersionIndex = i
			break
		}
	}

	if repoVersionIndex == -1 {
		return "", fmt.Errorf("could not find versioned module directory")
	}

	// Reconstruct the path up to the repo@version directory
	moduleRootParts := parts[:repoVersionIndex+1]
	moduleRoot := filepath.Join(absCachePath, filepath.Join(moduleRootParts...))

	return moduleRoot, nil
}
