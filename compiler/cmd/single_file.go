package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"compiler/config"
	"compiler/internal/ctx"
	"compiler/internal/frontend/parser"
	"compiler/internal/modules"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/collector"
	"compiler/internal/semantic/resolver"
	"compiler/internal/semantic/typecheck"
	"compiler/internal/symbol"
	"compiler/report"
)

// CompileSingleFile performs analysis on a single file without project context
// This is useful for LSP when analyzing standalone files or files not in the import tree
func CompileSingleFile(filePath string, isDebugEnabled bool) *ctx.CompilerContext {
	// Create a minimal project config for the single file
	fileDir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)

	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return nil
	}

	// Create a temporary project config for single-file analysis
	projectConfig := &config.ProjectConfig{
		Name:        "temp-single-file",
		ProjectRoot: fileDir,
		Compiler: config.CompilerConfig{
			Version: "0.0.1", // Use a compatible version
		},
		Build: config.BuildConfig{
			Entry:  fileName,
			Output: "temp",
		},
		Cache: config.CacheConfig{
			Path: ".ferret",
		},
		External: config.ExternalConfig{
			AllowSharing:        false,
			AllowRemoteImport:   false,
			AllowExternalImport: false,
		},
		Dependencies: config.DependencyConfig{
			Packages: make(map[string]string),
		},
		Neighbors: config.NeighborConfig{
			Projects: make(map[string]string),
		},
	}

	// Temporarily disable context creation restriction for single-file analysis
	context := createSingleFileContext(projectConfig)

	defer func() {
		if r := recover(); r != nil {
			// Don't panic in LSP mode, just capture the error
			if context != nil && context.Reports != nil {
				context.Reports.AddError("", nil, fmt.Sprintf("Internal compiler error: %v", r), report.PARSING_PHASE)
			}
		}
	}()

	// Parse the single file
	p := parser.NewParser(filePath, context, isDebugEnabled)
	program := p.Parse()

	if program == nil {
		return context
	}

	// Perform semantic analysis with limited scope
	anz := analyzer.NewAnalyzerNode(program, context, isDebugEnabled)

	// Collect symbols (will only include this file)
	collector.CollectSymbols(anz)

	// Resolve symbols (limited to built-ins and local symbols)
	resolver.ResolveProgram(anz)

	// Type checking
	typecheck.CheckProgram(anz)

	return context
}

// createSingleFileContext creates a compiler context without the singleton restriction
func createSingleFileContext(projectConfig *config.ProjectConfig) *ctx.CompilerContext {
	// This is a simplified version of ctx.NewCompilerContext that doesn't enforce singleton pattern
	entryPoint := filepath.Join(projectConfig.ProjectRoot, projectConfig.Build.Entry)
	entryPoint = filepath.ToSlash(entryPoint)

	remoteCachePath := filepath.Join(projectConfig.ProjectRoot, ".ferret")
	remoteCachePath = filepath.ToSlash(remoteCachePath)
	os.MkdirAll(remoteCachePath, 0755)

	// For single files, we don't need builtin modules discovery
	builtinModules := make(map[string]string)

	return &ctx.CompilerContext{
		EntryPoint:          entryPoint,
		Builtins:            symbol.AddPreludeSymbols(symbol.NewSymbolTable(nil)),
		Modules:             make(map[string]*modules.Module),
		Reports:             report.Reports{},
		ProjectConfig:       projectConfig,
		ProjectStack:        []*config.ProjectConfig{},
		RemoteCachePath:     remoteCachePath,
		BuiltinModules:      builtinModules,
		ProjectRootFullPath: projectConfig.ProjectRoot,
	}
}
