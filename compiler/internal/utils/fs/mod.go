package fs

import (
	"compiler/config"
	"os"
	"path/filepath"
	"strings"
)

// Check if file exists and is a regular file
func IsValidFile(filename string) bool {
	fileInfo, err := os.Stat(filepath.FromSlash(filename))
	return err == nil && fileInfo.Mode().IsRegular()
}

func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	return err == nil && fileInfo.Mode().IsDir()
}

func FirstPart(path string) string {
	if path == "" {
		return ""
	}

	// Normalize path separators to forward slashes for cross-platform consistency
	normalized := strings.ReplaceAll(path, "\\", "/")
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

	// Normalize path separators to forward slashes for cross-platform consistency
	normalized := strings.ReplaceAll(path, "\\", "/")
	normalized = strings.Trim(normalized, "/")

	parts := strings.Split(normalized, "/")

	if len(parts) > 0 && parts[len(parts)-1] != "" {
		// remove extension if present
		return strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
	}

	return ""
}

func DirectChilds(dirname string) (map[string]string, error) {
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}

	childs := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			// must contain a fer.ret to be considered a valid module
			rootLocation := filepath.Join(dirname, entry.Name())
			projectConfig, err := config.LoadProjectConfig(rootLocation)
			if err != nil {
				continue
			}
			childs[projectConfig.Name] = rootLocation
		}
	}

	return childs, nil
}
