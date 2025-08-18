package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"compiler/cmd/flags"
	"compiler/colors"
	"compiler/config"
	"compiler/internal/ctx"

	"compiler/internal/frontend/parser"

	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/collector"
	"compiler/internal/semantic/resolver"
	"compiler/internal/semantic/typecheck"
)

// checkCompilerVersion checks if the current compiler version is compatible
func checkCompilerVersion(requiredVersion string) error {
	currentVersion := flags.FERRET_VERSION
	// Parse versions (simple semantic version comparison)
	current := parseVersion(currentVersion)
	required := parseVersion(requiredVersion)

	// Check if current version is less than required
	if compareVersions(current, required) < 0 {
		return fmt.Errorf("compiler version %s is less than required version %s", currentVersion, requiredVersion)
	}

	return nil
}

// parseVersion parses a semantic version string into comparable parts
func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	nums := make([]int, 3) // major.minor.patch

	for i, part := range parts {
		if i >= 3 {
			break
		}
		if num, err := strconv.Atoi(part); err == nil {
			nums[i] = num
		}
	}

	return nums
}

// compareVersions compares two version arrays
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 []int) int {
	for i := range 3 {
		if v1[i] < v2[i] {
			return -1
		}
		if v1[i] > v2[i] {
			return 1
		}
	}
	return 0
}

// Compiler Does parse, analyze, and compile the source code.
func Compile(config *config.ProjectConfig, isDebugEnabled bool) (context *ctx.CompilerContext) {

	// Check if entry point file exists
	if _, err := os.Stat(config.ProjectRoot); err != nil {
		colors.RED.Printf("❌ Entry point file not found: %s\n", config.ProjectRoot)
		os.Exit(1)
	}

	// Check compiler version compatibility
	if err := checkCompilerVersion(config.Compiler.Version); err != nil {
		colors.RED.Printf("❌ Compiler version incompatibility: %s\n", err)
		os.Exit(1)
	}

	colors.BLUE.Printf("🚀 Running project with entry point: %s\n", config.Build.Entry)

	fullPath, err := filepath.Abs(filepath.Join(config.ProjectRoot, config.Build.Entry))
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path: %w", err))
	}

	fullPath = filepath.ToSlash(fullPath) // Ensure forward slashes for consistency

	context = ctx.NewCompilerContext(config)

	defer func() {
		context.Reports.DisplayAll()
		if r := recover(); r != nil {
			colors.ORANGE.Println("PANIC occurred:", r)
			fmt.Println("Stack trace:")
			debug.PrintStack()
		}
	}()

	p := parser.NewParser(fullPath, context, isDebugEnabled)
	program := p.Parse()

	if program == nil {
		colors.RED.Println("Failed to parse the program.")
		return context
	}

	if isDebugEnabled {
		colors.BLUE.Printf("---------- [Parsing done] ----------\n")
	}

	anz := analyzer.NewAnalyzerNode(program, context, isDebugEnabled)

	// --- Semantic Analysis ---
	// Collect symbols
	collector.CollectSymbols(anz)

	if isDebugEnabled {
		colors.BLUE.Printf("---------- [Symbol Collection done] ----------\n")
	}

	resolver.ResolveProgram(anz)

	if isDebugEnabled {
		colors.GREEN.Println("---------- [Resolver done] ----------")
	}

	typecheck.CheckProgram(anz)

	if context.Reports.HasErrors() {
		panic("Compilation stopped due to type checking errors")
	}

	if isDebugEnabled {
		colors.GREEN.Println("---------- [Type Checking done] ----------")
	}

	return context
}
