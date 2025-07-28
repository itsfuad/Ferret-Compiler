package registry

import (
	"compiler/internal/ctx"
	"path/filepath"
	"strings"
)

// IsFileInRemoteCache checks if a file is located in the remote module cache
func IsFileInRemoteCache(filePath string, ctx *ctx.CompilerContext) bool {
	if filePath == "" {
		return false
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	absCachePath, err := filepath.Abs(ctx.RemoteCachePath)
	if err != nil {
		return false
	}

	return strings.HasPrefix(absFilePath, absCachePath)
}

// GetRemoteModulePrefix extracts the GitHub repository prefix from a cached file path
// Example: /cache/github.com/user/repo@v1/data/file.fer -> github.com/user/repo
func GetRemoteModulePrefix(filePath string, ctx *ctx.CompilerContext) string {
	if !IsFileInRemoteCache(filePath, ctx) {
		return ""
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return ""
	}

	absCachePath, err := filepath.Abs(ctx.RemoteCachePath)
	if err != nil {
		return ""
	}

	// Get relative path within cache
	relPath, err := filepath.Rel(absCachePath, absFilePath)
	if err != nil {
		return ""
	}

	// Normalize to forward slashes
	relPath = filepath.ToSlash(relPath)

	// Extract repo prefix: github.com/user/repo@version -> github.com/user/repo
	parts := strings.Split(relPath, "/")
	if len(parts) >= 3 {
		// Take first 3 parts and remove version from repo name
		if strings.Contains(parts[2], "@") {
			// Remove version suffix from repo name
			repoParts := strings.Split(parts[2], "@")
			return parts[0] + "/" + parts[1] + "/" + repoParts[0]
		}
	}

	return ""
}
