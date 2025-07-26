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
const INVALID_GITHUB_PATH_MSG = "invalid GitHub repository path: %s"

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
		return resolveRemoteModule(importPath, ctxx)
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

// resolveRemoteModule implements the new import system workflow
func resolveRemoteModule(importPath string, ctxx *ctx.CompilerContext) (string, error) {
	repoPath, requestedVersion, subPath := ctxx.ParseRemoteImport(importPath)
	if repoPath == "" {
		return "", fmt.Errorf("invalid remote import path: %s", importPath)
	}
	if err := checkRemoteImportsEnabled(ctxx); err != nil {
		return "", err
	}

	targetVersion, err := getTargetVersion(repoPath, requestedVersion, ctxx.ProjectRoot)
	if err != nil {
		return "", err
	}

	lockFile, err := registry.LoadLockFile(ctxx.ProjectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read ferret.lock: %w", err)
	}

	actualVersion, err := resolveVersionConstraint(repoPath, targetVersion, lockFile, ctxx)
	if err != nil {
		return "", fmt.Errorf("failed to resolve version for %s@%s: %w", repoPath, targetVersion, err)
	}

	if err := ensureModuleCached(repoPath, actualVersion, ctxx); err != nil {
		return "", err
	}

	if err := registry.ValidateModuleSharing(ctxx, repoPath, actualVersion); err != nil {
		return "", fmt.Errorf("cached module validation failed: %w", err)
	}

	return resolveCachedModulePathFlat(repoPath+"@"+actualVersion, subPath, ctxx)
}

func checkRemoteImportsEnabled(ctxx *ctx.CompilerContext) error {
	if ctxx.ProjectConfig != nil && !ctxx.ProjectConfig.Remote.Enabled {
		return fmt.Errorf(`remote module imports are disabled in this project

To enable remote imports, set 'enabled = true' in the [remote] section of fer.ret:

[remote]
enabled = true
share = false`)
	}
	return nil
}

func getTargetVersion(repoPath, requestedVersion, projectRoot string) (string, error) {
	dependencies, err := registry.ParseFerRetDependencies(projectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read fer.ret: %w", err)
	}

	declaredVersion, isDeclared := dependencies[repoPath]
	if !isDeclared {
		return "", fmt.Errorf("module '%s' is not installed.\n To install this module, run:\n  ferret get %s", repoPath, repoPath)
	}

	if requestedVersion != "" && requestedVersion != "latest" && requestedVersion != declaredVersion {
		colors.YELLOW.Printf("Warning: Requested version %s differs from declared version %s for %s, using declared version\n",
			requestedVersion, declaredVersion, repoPath)
	}

	return declaredVersion, nil
}

func ensureModuleCached(repoPath, actualVersion string, ctxx *ctx.CompilerContext) error {
	flatModuleName := repoPath + "@" + actualVersion
	if ctxx.IsRemoteModuleCachedFlat(flatModuleName) {
		return nil
	}

	lockEntry, err := registry.GetLockedEntryFlat(ctxx.ProjectRoot, flatModuleName)
	action := "Installing declared module"
	if err == nil && lockEntry != nil {
		action = "Module was previously installed but cache is missing. Re-downloading"
	}
	colors.CYAN.Printf("%s %s...\n", action, flatModuleName)
	return autoInstallModule(repoPath, actualVersion, ctxx)
}

func resolveVersionConstraint(repoPath, constraint string, lockFile *registry.LockFile, ctxx *ctx.CompilerContext) (string, error) {
	if isExactVersion(constraint) {
		return validateExactVersion(repoPath, constraint)
	}
	if cached := findCurrentCachedVersion(repoPath, ctxx); cached != "" {
		return resolveFromCache(repoPath, constraint, cached, ctxx)
	}
	if v := findInLockFile(repoPath, constraint, lockFile); v != "" {
		return v, nil
	}
	if constraint == "latest" {
		return resolveLatestVersion(repoPath)
	}
	return findBestVersionForConstraint(repoPath, constraint)
}

func isExactVersion(v string) bool {
	return !strings.HasPrefix(v, "^") && !strings.HasPrefix(v, "~") && v != "latest"
}

func validateExactVersion(repoPath, version string) (string, error) {
	if strings.HasPrefix(repoPath, REMOTE_HOST) {
		if err := validateGitHubVersionExists(repoPath, version); err != nil {
			return "", fmt.Errorf("version validation failed for %s@%s: %w", repoPath, version, err)
		}
	}
	return version, nil
}

func resolveFromCache(repoPath, constraint, cached string, ctxx *ctx.CompilerContext) (string, error) {
	if versionSatisfiesConstraint(cached, constraint) {
		colors.GREEN.Printf("Current cached version %s satisfies constraint %s for %s\n", cached, constraint, repoPath)
		return cached, nil
	}
	colors.YELLOW.Printf("Current cached version %s does not satisfy new constraint %s for %s\n", cached, constraint, repoPath)
	newVersion, err := findBestVersionForConstraint(repoPath, constraint)
	if err != nil {
		return "", fmt.Errorf("failed to find version satisfying constraint %s for %s: %w", constraint, repoPath, err)
	}
	if newVersion != cached {
		colors.CYAN.Printf("Version change detected: %s -> %s for %s\n", cached, newVersion, repoPath)
		if err := handleVersionChange(repoPath, cached, newVersion, ctxx); err != nil {
			return "", fmt.Errorf("failed to handle version change for %s: %w", repoPath, err)
		}
	}
	return newVersion, nil
}

func findInLockFile(repoPath, constraint string, lockFile *registry.LockFile) string {
	for flatName := range lockFile.Packages {
		if strings.HasPrefix(flatName, repoPath+"@") {
			actualVersion := strings.TrimPrefix(flatName, repoPath+"@")
			if versionSatisfiesConstraint(actualVersion, constraint) {
				return actualVersion
			}
		}
	}
	return ""
}

func findCurrentCachedVersion(repoPath string, ctxx *ctx.CompilerContext) string {
	cacheDir := ctxx.RemoteCachePath
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return ""
	}

	prefix := repoPath + "@"
	var foundVersion string

	filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(cacheDir, path)
		if err != nil {
			return nil
		}
		if strings.HasPrefix(filepath.ToSlash(relPath), prefix) {
			version := strings.TrimPrefix(filepath.ToSlash(relPath), prefix)
			if version != "" && !strings.Contains(version, "/") {
				foundVersion = version
				return filepath.SkipDir
			}
		}
		return nil
	})
	return foundVersion
}

func versionSatisfiesConstraint(version, constraint string) bool {
	switch {
	case constraint == "latest":
		return true
	case constraint == version:
		return true
	case strings.HasPrefix(constraint, "^"):
		return satisfiesCaret(version, strings.TrimPrefix(constraint, "^"))
	case strings.HasPrefix(constraint, "~"):
		return satisfiesTilde(version, strings.TrimPrefix(constraint, "~"))
	default:
		return false
	}
}

func satisfiesCaret(version, base string) bool {
	if strings.HasPrefix(base, "v") && strings.Contains(base, ".") {
		major := strings.SplitN(base, ".", 2)[0]
		return strings.HasPrefix(version, major+".")
	}
	return strings.HasPrefix(version, base)
}

func satisfiesTilde(version, base string) bool {
	if !strings.HasPrefix(base, "v") {
		return strings.HasPrefix(version, base)
	}
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return strings.HasPrefix(version, base)
	}
	majorMinor := parts[0] + "." + parts[1]
	return strings.HasPrefix(version, majorMinor+".")
}

// autoInstallModule automatically installs a module based on fer.ret and lock file data
func autoInstallModule(repoPath, version string, ctxx *ctx.CompilerContext) error {
	// This will trigger the download/installation process
	// We'll implement this to call the existing remote installation logic
	colors.GREEN.Printf("Auto-installing %s@%s from declared dependencies...\n", repoPath, version)

	// Call the download function but don't update fer.ret since it's already declared
	err := registry.DownloadRemoteModuleWithoutFerRetUpdate(ctxx, repoPath, version)
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

// findBestVersionForConstraint finds the best version that satisfies a constraint
func findBestVersionForConstraint(repoPath, constraint string) (string, error) {
	if !strings.HasPrefix(repoPath, REMOTE_HOST) {
		return "", fmt.Errorf("version constraint resolution only supported for GitHub repositories")
	}

	// Get all available versions from GitHub
	parts := strings.Split(strings.TrimPrefix(repoPath, REMOTE_HOST), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf(INVALID_GITHUB_PATH_MSG, repoPath)
	}

	owner, repo := parts[0], parts[1]
	availableVersions, err := getAllAvailableVersionsFromGitHub(owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get available versions: %w", err)
	}

	// Find the best version that satisfies the constraint
	for _, version := range availableVersions {
		if versionSatisfiesConstraint(version, constraint) {
			return version, nil
		}
	}

	return "", fmt.Errorf("no version found that satisfies constraint %s", constraint)
}

// handleVersionChange removes old cached version and prepares for new version download
func handleVersionChange(repoPath, oldVersion, newVersion string, ctxx *ctx.CompilerContext) error {
	colors.CYAN.Printf("Handling version change for %s: %s -> %s\n", repoPath, oldVersion, newVersion)

	// Remove old cached version
	oldFlatName := repoPath + "@" + oldVersion
	oldCachePath := ctxx.GetRemoteModuleCachePathFlat(oldFlatName)

	if _, err := os.Stat(oldCachePath); err == nil {
		colors.YELLOW.Printf("Removing old cached version: %s\n", oldFlatName)
		err := os.RemoveAll(oldCachePath)
		if err != nil {
			return fmt.Errorf("failed to remove old cache for %s: %w", oldFlatName, err)
		}
	}

	// The new version will be downloaded by the calling logic
	colors.GREEN.Printf("Prepared for version change: %s -> %s\n", oldVersion, newVersion)
	return nil
}

// validateGitHubVersionExists validates that a version exists on GitHub
func validateGitHubVersionExists(repoPath, version string) error {
	parts := strings.Split(strings.TrimPrefix(repoPath, REMOTE_HOST), "/")
	if len(parts) < 2 {
		return fmt.Errorf(INVALID_GITHUB_PATH_MSG, repoPath)
	}

	owner, repo := parts[0], parts[1]

	// Use the registry function to validate
	return registry.ValidateGitHubVersion(owner, repo, version)
}

// resolveLatestVersion resolves "latest" to the actual latest version
func resolveLatestVersion(repoPath string) (string, error) {
	if !strings.HasPrefix(repoPath, REMOTE_HOST) {
		return "", fmt.Errorf("latest version resolution only supported for GitHub repositories")
	}

	parts := strings.Split(strings.TrimPrefix(repoPath, REMOTE_HOST), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf(INVALID_GITHUB_PATH_MSG, repoPath)
	}

	owner, repo := parts[0], parts[1]
	_, version, err := registry.GetLatestGitHubRelease(owner, repo)
	return version, err
}

// getAllAvailableVersionsFromGitHub gets all available versions for a GitHub repository
func getAllAvailableVersionsFromGitHub(owner, repo string) ([]string, error) {
	return registry.GetAllAvailableVersions(owner, repo)
}
