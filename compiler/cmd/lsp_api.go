package cmd

import (
	"compiler/config"
	"compiler/internal/ctx"
	"compiler/report"
)

// CompilerResult wraps the compilation result for external use
type CompilerResult struct {
	Reports report.Reports
	Success bool
}

// CompileForLSP performs compilation and returns results for LSP use
func CompileForLSP(filePath string, isDebugEnabled bool) *CompilerResult {
	var context *ctx.CompilerContext

	defer func() {
		if context != nil {
			context.Destroy()
		}
	}()

	context = CompileSingleFile(filePath, isDebugEnabled)
	if context != nil {
		return &CompilerResult{
			Reports: context.Reports,
			Success: !context.Reports.HasErrors(),
		}
	}

	return &CompilerResult{
		Reports: make(report.Reports, 0),
		Success: false,
	}
}

// CompileProjectForLSP performs project-based compilation for LSP
func CompileProjectForLSP(projectRoot string, isDebugEnabled bool) *CompilerResult {
	var context *ctx.CompilerContext

	defer func() {
		if context != nil {
			context.Destroy()
		}
	}()

	// Load project config and compile
	if conf, err := config.LoadProjectConfig(projectRoot); err == nil {
		context = Compile(conf, isDebugEnabled)
		if context != nil {
			return &CompilerResult{
				Reports: context.Reports,
				Success: !context.Reports.HasErrors(),
			}
		}
	}

	return &CompilerResult{
		Reports: make(report.Reports, 0),
		Success: false,
	}
}
