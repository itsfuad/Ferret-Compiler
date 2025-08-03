package modules

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/symbol"
	"ferret/compiler/internal/utils/fs"
	"path/filepath"
	"strings"
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

type Module struct {
	AST         *ast.Program
	SymbolTable *symbol.SymbolTable
	Phase       ModulePhase // Current processing phase
	IsBuiltin   bool        // Whether this is a builtin module
	Type        ModuleType
}

// ModuleType represents the category of a module
type ModuleType int

const (
	UNKNOWN ModuleType = iota
	LOCAL
	BUILTIN
	REMOTE
)

func (mt ModuleType) String() string {
	switch mt {
	case LOCAL:
		return "LOCAL"
	case BUILTIN:
		return "BUILTIN"
	case REMOTE:
		return "REMOTE"
	default:
		return "UNKNOWN"
	}
}

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

const REMOTE_HOST = "github.com/"

func IsRemote(importPath string) bool {
	return strings.HasPrefix(importPath, REMOTE_HOST)
}

func IsBuiltinModule(importRoot string) bool {
	return BUILTIN_MODULES[importRoot]
}

// GetModuleType categorizes an import path
func GetModuleType(importPath string, projectName string) ModuleType {
	importRoot := fs.FirstPart(importPath)

	if IsRemote(importPath) {
		return REMOTE
	}

	if IsBuiltinModule(importRoot) {
		return BUILTIN
	}

	if importRoot == projectName {
		return LOCAL
	}

	// Default to local for unrecognized paths
	return UNKNOWN
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
