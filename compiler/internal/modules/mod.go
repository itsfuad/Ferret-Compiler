package modules

import (
	"path/filepath"
	"strings"

	"compiler/constants"
	"compiler/internal/frontend/ast"
	"compiler/internal/symbol"
	"compiler/internal/utils/fs"
)

// ModulePhase represents the current processing phase of a module
type ModulePhase int

const (
	PHASE_NOT_STARTED ModulePhase = iota
	PHASE_PARSED                  // Module has been parsed into AST
	PHASE_COLLECTED               // Symbols have been collected
	PHASE_RESOLVED                // Symbols have been resolved
	PHASE_TYPECHECKED             // Type checking completed
)

func (p ModulePhase) String() string {
	switch p {
	case PHASE_NOT_STARTED:
		return "Not Started"
	case PHASE_PARSED:
		return "Parsed"
	case PHASE_COLLECTED:
		return "Collected"
	case PHASE_RESOLVED:
		return "Resolved"
	case PHASE_TYPECHECKED:
		return "Type Checked"
	default:
		return "Unknown"
	}
}

// ModuleType represents the category of a module
type ModuleType int

const (
	UNKNOWN ModuleType = iota
	LOCAL
	BUILTIN
	REMOTE
	NEIGHBOR // External neighboring project (like Go's replace directive)
)

func (mt ModuleType) String() string {
	switch mt {
	case LOCAL:
		return "LOCAL"
	case BUILTIN:
		return "BUILTIN"
	case REMOTE:
		return "REMOTE"
	case NEIGHBOR:
		return "NEIGHBOR"
	default:
		return "UNKNOWN"
	}
}

type Module struct {
	AST         *ast.Program
	SymbolTable *symbol.SymbolTable
	Phase       ModulePhase     // Current processing phase
	UsedImports map[string]bool // Track which imports are used in this file
}

func IsRemote(importPath string) bool {
	return strings.HasPrefix(importPath, constants.GITHUB_HOST)
}

// IsLocalProject checks if an import path refers to a local project defined in locals config
func IsLocalProject(importPath string, localsConfig map[string]string) bool {
	if localsConfig == nil {
		return false
	}

	importRoot := fs.FirstPart(importPath)
	_, exists := localsConfig[importRoot]
	return exists
}

// GetRemoteModulePrefix extracts the GitHub repository prefix from a cached file path
// Example: /cache/github.com/user/repo@v1/data/file.fer -> github.com/user/repo
func GetRemoteModulePrefix(filePath string, cachePath string) string {
	if !IsFileInRemoteCache(filePath, cachePath) {
		return ""
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return ""
	}

	absCachePath, err := filepath.Abs(cachePath)
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

// IsFileInRemoteCache checks if a file is located in the remote module cache
func IsFileInRemoteCache(filePath string, cachePath string) bool {
	if filePath == "" {
		return false
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	absCachePath, err := filepath.Abs(cachePath)
	if err != nil {
		return false
	}

	return strings.HasPrefix(absFilePath, absCachePath)
}
