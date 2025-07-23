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
		return "", fmt.Errorf("remote imports are not supported yet: %s", importPath)
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
