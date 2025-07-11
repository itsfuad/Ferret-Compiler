package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"compiler/ctx"
)

const EXT = ".fer"
const REMOTE_HOST = "github.com/"

func IsRemote(importPath string) bool {
	return strings.HasPrefix(importPath, REMOTE_HOST)
}

// Check if file exists and is a regular file
func IsValidFile(filename string) bool {
	fileInfo, err := os.Stat(filename)
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
	// Normalize all separators to OS-specific
	clean := filepath.ToSlash(path) // Use ToSlash for uniform splitting
	parts := strings.Split(clean, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func LastPart(path string) string {
	// Normalize all separators to OS-specific
	clean := filepath.ToSlash(path) // Use ToSlash for uniform splitting
	parts := strings.Split(clean, "/")
	if len(parts) > 0 {
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

	projectRoot := LastPart(ctxx.ProjectRoot)
	if projectRoot == "" {
		return "", fmt.Errorf("invalid project root: %s", ctxx.ProjectRoot)
	}

	if importRoot == projectRoot {
		resolvedPath := filepath.Join(strings.TrimSuffix(ctxx.ProjectRoot, projectRoot), importPath+EXT)
		if IsValidFile(resolvedPath) {
			return resolvedPath, nil
		}
		return "", fmt.Errorf("module not found: %s", importPath)
	}

	return "", fmt.Errorf("external imports are not supported yet: %s", importPath)
}
