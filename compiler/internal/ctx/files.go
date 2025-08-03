package ctx

import (
	"ferret/compiler/internal/modules"
	"fmt"
)

const EXT = ".fer"

const INVALID_GITHUB_PATH_MSG = "invalid GitHub repository path: %s"

// ResolveModuleLocation resolves the full path of a module based on its import path
// It returns the physical location of the module file on disk
// or an error if the module cannot be found.
func ResolveModuleLocation(importPath, currentFileFullPath string, ctxx *CompilerContext) (string, error) {
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
	moduleType := modules.GetModuleType(importPath, projectName)

	// Handle special case: if current file is in remote cache, imports should be in remote context
	if modules.IsFilepathInCache(currentFileFullPath, ctxx.RemoteCachePath) {
		return modules.ResolveModuleInRemoteContext(importPath, currentFileFullPath, ctxx.ProjectRoot, ctxx.RemoteCachePath)
	}

	// Route to appropriate resolver based on module type
	switch moduleType {
	case modules.REMOTE:
		return modules.ResolveRemoteModule(importPath, ctxx.ProjectRoot, ctxx.RemoteCachePath, currentFileFullPath)
	case modules.BUILTIN:
		return modules.ResolveBuiltinModule(importPath, ctxx.ModulesPath)
	case modules.LOCAL:
		return modules.ResolveLocalModule(importPath, projectName, ctxx.ProjectRoot)
	default:
		return "", fmt.Errorf("unknown module type for import: %s", importPath)
	}
}
