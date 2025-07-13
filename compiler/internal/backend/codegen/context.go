package codegen

import (
	"compiler/ctx"
	"fmt"
)

// CodeGenContext holds context information during code generation
type CodeGenContext struct {
	CompilerContext *ctx.CompilerContext
	CurrentModule   string
	CurrentFunction string
	LabelCounter    int
	LocalVariables  map[string]VariableInfo
	Parameters      map[string]VariableInfo
}

// VariableInfo holds information about variables for code generation
type VariableInfo struct {
	Name     string
	Offset   int  // Stack offset for local variables
	Size     int  // Size in bytes
	IsGlobal bool // Whether the variable is global
	IsParam  bool // Whether the variable is a function parameter
}

// NewCodeGenContext creates a new code generation context
func NewCodeGenContext(compilerCtx *ctx.CompilerContext) *CodeGenContext {
	return &CodeGenContext{
		CompilerContext: compilerCtx,
		LabelCounter:    0,
		LocalVariables:  make(map[string]VariableInfo),
		Parameters:      make(map[string]VariableInfo),
	}
}

// GetNextLabel generates a unique label with the given prefix
func (ctx *CodeGenContext) GetNextLabel(prefix string) string {
	ctx.LabelCounter++
	return fmt.Sprintf("%s_%d", prefix, ctx.LabelCounter)
}

// AddLocalVariable adds a local variable to the context
func (ctx *CodeGenContext) AddLocalVariable(name string, size int, offset int) {
	ctx.LocalVariables[name] = VariableInfo{
		Name:     name,
		Offset:   offset,
		Size:     size,
		IsGlobal: false,
		IsParam:  false,
	}
}

// AddParameter adds a function parameter to the context
func (ctx *CodeGenContext) AddParameter(name string, size int, offset int) {
	ctx.Parameters[name] = VariableInfo{
		Name:     name,
		Offset:   offset,
		Size:     size,
		IsGlobal: false,
		IsParam:  true,
	}
}

// GetVariable retrieves variable information
func (ctx *CodeGenContext) GetVariable(name string) (VariableInfo, bool) {
	// Check parameters first
	if info, exists := ctx.Parameters[name]; exists {
		return info, true
	}

	// Then check local variables
	if info, exists := ctx.LocalVariables[name]; exists {
		return info, true
	}

	// If not found locally, it might be a global variable
	return VariableInfo{
		Name:     name,
		IsGlobal: true,
	}, false
}
