package fs

import (
	"os"
	"path/filepath"
	"strings"
)

// Check if file exists and is a regular file
func IsValidFile(filename string) bool {
	fileInfo, err := os.Stat(filepath.FromSlash(filename))
	return err == nil && fileInfo.Mode().IsRegular()
}

func FirstPart(path string) string {
	if path == "" {
		return ""
	}

	normalized := filepath.ToSlash(path) // Ensure forward slashes for consistency
	//remove leading and trailing slashes
	normalized = strings.Trim(normalized, "/")

	parts := strings.Split(normalized, "/")

	if len(parts) > 0 && parts[0] != "" {
		// remove extension if present
		return strings.TrimSuffix(parts[0], filepath.Ext(parts[0]))
	}
	return ""
}

func LastPart(path string) string {
	if path == "" {
		return ""
	}

	normalized := filepath.ToSlash(path) // Ensure forward slashes for consistency
	normalized = strings.Trim(normalized, "/")

	parts := strings.Split(normalized, "/")

	if len(parts) > 0 && parts[len(parts)-1] != "" {
		// remove extension if present
		return strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
	}

	return ""
}
