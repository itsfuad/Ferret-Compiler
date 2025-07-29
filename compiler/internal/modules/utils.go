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
