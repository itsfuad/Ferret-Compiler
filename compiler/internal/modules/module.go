package modules

import (
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
)

type Module struct {
	AST         *ast.Program
	SymbolTable *ctx.SymbolTable
	Phase       ModulePhase // Current processing phase
	IsBuiltin   bool        // Whether this is a builtin module
	Type        ModuleType
}

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

// ModuleType represents the category of a module
type ModuleType int

const (
	LOCAL ModuleType = iota
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

func IsBuiltinModule(importRoot string) bool {
	return BUILTIN_MODULES[importRoot]
}