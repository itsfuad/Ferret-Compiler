package modules

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/symbol"
	"compiler/internal/utils/fs"
	"path/filepath"
	"strings"
)

// ModulePhase represents the current processing phase of a module
type ModulePhase int

const (
	PhaseNotStarted  ModulePhase = iota
	PhaseParsed                  // Module has been parsed into AST
	PhaseCollected               // Symbols have been collected
	PhaseResolved                // Symbols have been resolved
	PhaseTypeChecked             // Type checking completed
)

func (p ModulePhase) String() string {
	switch p {
	case PhaseNotStarted:
		return "Not Started"
	case PhaseParsed:
		return "Parsed"
	case PhaseCollected:
		return "Collected"
	case PhaseResolved:
		return "Resolved"
	case PhaseTypeChecked:
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
