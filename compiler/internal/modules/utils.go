package modules

import (
	"fmt"
	"strings"
)

// SplitRemotePath extracts repo and version information from a remote import path
// Example: "github.com/user/repo@v1.0.0" -> ("github.com/user/repo", "v1.0.0", "user/repo")
func SplitRemotePath(importPath string) (repoPath, version, repoName string, err error) {
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

// NormalizeVersion ensures all versions have the "v" prefix for consistency
// Examples: "1.0.0" -> "v1.0.0", "v1.0.0" -> "v1.0.0", "latest" -> "latest"
func NormalizeVersion(version string) string {
	if version == "" || version == "latest" {
		return version
	}

	// If it already has "v" prefix, return as-is
	if strings.HasPrefix(version, "v") {
		return version
	}

	// Add "v" prefix if it looks like a semantic version
	if isSemanticVersion(version) {
		return "v" + version
	}

	return version
}

// StripVersionPrefix removes the "v" prefix from version for GitHub API compatibility
// Examples: "v1.0.0" -> "1.0.0", "1.0.0" -> "1.0.0"
func StripVersionPrefix(version string) string {
	if strings.HasPrefix(version, "v") {
		return strings.TrimPrefix(version, "v")
	}
	return version
}

// isSemanticVersion checks if a string looks like a semantic version
func isSemanticVersion(version string) bool {
	// Simple check for X.Y.Z pattern (with optional additional parts)
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	// Check if all parts are numeric-like (allowing pre-release suffixes)
	for _, part := range parts {
		if part == "" {
			return false
		}
		// First character should be a digit
		if len(part) > 0 && (part[0] < '0' || part[0] > '9') {
			return false
		}
	}

	return true
}
