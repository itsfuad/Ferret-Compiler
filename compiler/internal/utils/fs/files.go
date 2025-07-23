package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"compiler/colors"
	"compiler/internal/ctx"
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

	if IsRemote(importPath) {
		return resolveRemoteModule(importPath, ctxx)
	}

	//the first part of the import path is the root
	importRoot := FirstPart(importPath)
	if importRoot == "" {
		return "", fmt.Errorf("invalid import path: %s", importPath)
	}

	// Check if it's a built-in module
	if IsBuiltinModule(importRoot) {
		// Search for the module in the system modules directory
		// e.g., "std/io" becomes "modules/std/io.fer"
		modulePath := filepath.Join(ctxx.ModulesPath, importPath+EXT)
		colors.AQUA.Printf("Searching for built-in module: %s -> %s\n", importPath, modulePath)
		if IsValidFile(modulePath) {
			return modulePath, nil
		}
		return "", fmt.Errorf("built-in module not found: %s", importPath)
	}

	// Use project name from configuration instead of folder name
	projectName := ctxx.ProjectConfig.Name
	if projectName == "" {
		return "", fmt.Errorf("project name not defined in configuration")
	}

	if importRoot == projectName {
		// Remove the project name from the import path and resolve relative to project root
		// e.g., "myapp/maths/math" becomes "maths/math"
		relativePath := strings.TrimPrefix(importPath, projectName+"/")
		resolvedPath := filepath.Join(ctxx.ProjectRoot, relativePath+EXT)

		if IsValidFile(resolvedPath) {
			return resolvedPath, nil
		}
		return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
	}

	return "", fmt.Errorf("module `%s` does not exist in this project", importPath)
}

// resolveRemoteModule resolves a remote module import by checking local cache
func resolveRemoteModule(importPath string, ctxx *ctx.CompilerContext) (string, error) {
	repoPath, version, subPath := ctxx.ParseRemoteImport(importPath)

	if repoPath == "" {
		return "", fmt.Errorf("invalid remote import path: %s", importPath)
	}

	// Resolve the version to the actual cached version
	actualVersion, err := resolveVersionForCache(repoPath, version, ctxx)
	if err != nil {
		return "", err
	}

	// Check if module is cached locally with the resolved version
	if !ctxx.IsRemoteModuleCached(repoPath, actualVersion) {
		return "", fmt.Errorf("remote module not found in cache: %s@%s\nRun: ferret get %s", repoPath, version, repoPath)
	}

	// Construct path to the cached module file
	cachePath := ctxx.GetRemoteModuleCachePath(repoPath, actualVersion)
	var modulePath string

	if subPath != "" {
		modulePath = filepath.Join(cachePath, subPath+EXT)
	} else {
		// If no subpath, look for main module file (could be index.fer, main.fer, etc.)
		// Try common entry point names
		possibleFiles := []string{"index.fer", "main.fer", "mod.fer"}
		for _, fileName := range possibleFiles {
			candidatePath := filepath.Join(cachePath, fileName)
			if IsValidFile(candidatePath) {
				modulePath = candidatePath
				break
			}
		}

		if modulePath == "" {
			return "", fmt.Errorf("no entry point found in cached module: %s@%s", repoPath, actualVersion)
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
	// List all cached versions for this repo
	cacheDir := filepath.Join(ctxx.RemoteCachePath, strings.TrimPrefix(repoPath, "github.com/"))

	// Look for directories that match the pattern reponame@version
	parentDir := filepath.Dir(cacheDir)
	repoName := filepath.Base(cacheDir)

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
