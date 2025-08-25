package modules

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/symbol"
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
