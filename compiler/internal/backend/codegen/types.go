package codegen

import (
	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
)

// Target represents different target architectures
type Target int

const (
	TargetX86_64 Target = iota
	TargetARM64
	TargetRISCV64
)

// GeneratorOptions holds configuration for code generation
type GeneratorOptions struct {
	OutputFile    string
	OptimizeLevel int
	DebugInfo     bool
	Target        Target
}

// CodeGenerator is the interface that all code generators must implement
type CodeGenerator interface {
	Generate(program *ast.Program, compilerCtx *ctx.CompilerContext) (string, error)
	GetTarget() Target
	SetOptions(options map[string]interface{})
}

// NewCodeGenerator creates a new code generator for the specified target
func NewCodeGenerator(target Target, options *GeneratorOptions) CodeGenerator {
	switch target {
	// Later, we can add more targets like ARM64, RISCV64, etc.
	default:
		return NewX8664Generator(options) // Default to x86-64
	}
}
