package modules

import (
	"fmt"
	"strings"
)

func ExtractRepoPathFromImport(importPath string) (string, error) {
	// Convert: github.com/owner/repo/folderA/folderB/file -> github.com/owner/repo
	parts := strings.Split(importPath, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid import path %q to extract repo", importPath)
	}
	return fmt.Sprintf("%s/%s/%s", parts[0], parts[1], parts[2]), nil
}

func ExtractModuleFromImport(importPath string) (string, error) {
	parts := strings.Split(importPath, "/")
	// "github.com/itsfuad/ferret-remote-mod/extern"
	// Need: host / owner / repo / folderA / folderB...
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid import path %q to extract module", importPath)
	}
	return strings.Join(parts[3:], "/"), nil
}
