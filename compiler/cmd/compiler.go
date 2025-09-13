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
	"compiler/constants"
	"compiler/internal/ctx"
	"compiler/internal/modules"

	"compiler/internal/frontend/parser"

	"compiler/internal/semantic/analyzer"
	"compiler/internal/semantic/collector"
	"compiler/internal/semantic/resolver"
	"compiler/internal/semantic/typecheck"
)

// VersionCheckResult contains information about a version check
type VersionCheckResult struct {
	PackageName     string
	PackagePath     string
	RequiredVersion string
	PackageType     string // "current project", "remote dependency", "neighbor project"
	IsCompatible    bool
	Error           error
}

// checkCompilerVersion checks if the current compiler version is compatible with a requirement
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

// checkAllPackageVersions performs comprehensive version checking across all packages
func checkAllPackageVersions(projectConfig *config.ProjectConfig) error {
	currentVersion := flags.FERRET_VERSION
	var incompatiblePackages []VersionCheckResult

	// Check current project
	if result := checkSinglePackageVersion(projectConfig, "current project", projectConfig.ProjectRoot); !result.IsCompatible {
		incompatiblePackages = append(incompatiblePackages, result)
	}

	// Check remote dependencies
	if remoteResults := checkRemoteDependencies(projectConfig, currentVersion); len(remoteResults) > 0 {
		incompatiblePackages = append(incompatiblePackages, remoteResults...)
	}

	// Check neighbor projects
	if neighborResults := checkNeighborProjects(projectConfig, currentVersion); len(neighborResults) > 0 {
		incompatiblePackages = append(incompatiblePackages, neighborResults...)
	}

	// Report all incompatible packages
	if len(incompatiblePackages) > 0 {
		return formatVersionCompatibilityError(currentVersion, incompatiblePackages)
	}

	return nil
}

// checkSinglePackageVersion checks a single package's version requirement
func checkSinglePackageVersion(projectConfig *config.ProjectConfig, packageType, packagePath string) VersionCheckResult {
	result := VersionCheckResult{
		PackageName:     projectConfig.Name,
		PackagePath:     packagePath,
		RequiredVersion: projectConfig.Compiler.Version,
		PackageType:     packageType,
		IsCompatible:    true,
	}

	if projectConfig.Compiler.Version != "" {
		if err := checkCompilerVersion(projectConfig.Compiler.Version); err != nil {
			result.IsCompatible = false
			result.Error = err
		}
	}

	return result
}

// checkRemoteDependencies checks version requirements for all remote dependencies
func checkRemoteDependencies(projectConfig *config.ProjectConfig, currentVersion string) []VersionCheckResult {
	var incompatiblePackages []VersionCheckResult

	// Load the dependency manager to get cached dependencies
	_, err := modules.NewDependencyManager(projectConfig.ProjectRoot)
	if err != nil {
		// If we can't load the dependency manager, we can't check remote deps
		// This is not a fatal error - just means no remote deps to check
		return incompatiblePackages
	}

	// Get the lockfile to find all installed dependencies
	lockfile, err := modules.LoadLockfile(projectConfig.ProjectRoot)
	if err != nil {
		return incompatiblePackages
	}

	// Check each dependency's config file for version requirements
	for depKey := range lockfile.Dependencies {
		if depPath := getRemoteDependencyPath(projectConfig, depKey); depPath != "" {
			if result := checkRemoteDependencyVersion(depKey, depPath, currentVersion); !result.IsCompatible {
				incompatiblePackages = append(incompatiblePackages, result)
			}
		}
	}

	return incompatiblePackages
}

// getRemoteDependencyPath constructs the path to a remote dependency
func getRemoteDependencyPath(projectConfig *config.ProjectConfig, depKey string) string {
	// Parse dependency key format: "host/owner/repo@version"
	parts := strings.Split(depKey, "@")
	if len(parts) != 2 {
		return ""
	}

	repoPath := parts[0]
	version := parts[1]

	// Split repo path: "host/owner/repo"
	repoParts := strings.Split(repoPath, "/")
	if len(repoParts) != 3 {
		return ""
	}

	host, owner, repo := repoParts[0], repoParts[1], repoParts[2]

	// Construct cache path
	cachePath := filepath.Join(projectConfig.ProjectRoot, constants.CACHE_DIR)
	return filepath.Join(cachePath, host, owner, modules.BuildPackageSpec(repo, version))
}

// checkRemoteDependencyVersion checks version requirement for a single remote dependency
func checkRemoteDependencyVersion(depKey, depPath, currentVersion string) VersionCheckResult {
	result := VersionCheckResult{
		PackageName:  depKey,
		PackagePath:  depPath,
		PackageType:  "remote dependency",
		IsCompatible: true,
	}

	// Try to load the dependency's config
	depConfig, err := config.LoadProjectConfig(depPath)
	if err != nil {
		// If we can't load the config, we can't check the version requirement
		// This might not be an error - the dependency might not have a config
		return result
	}

	result.PackageName = depConfig.Name
	result.RequiredVersion = depConfig.Compiler.Version

	if depConfig.Compiler.Version != "" {
		if err := checkCompilerVersion(depConfig.Compiler.Version); err != nil {
			result.IsCompatible = false
			result.Error = err
		}
	}

	return result
}

// checkNeighborProjects checks version requirements for all neighbor projects
func checkNeighborProjects(projectConfig *config.ProjectConfig, currentVersion string) []VersionCheckResult {
	var incompatiblePackages []VersionCheckResult

	// Check each neighbor project
	for neighborName, neighborPath := range projectConfig.Neighbors.Projects {
		// Resolve absolute path
		var absPath string
		if filepath.IsAbs(neighborPath) {
			absPath = neighborPath
		} else {
			absPath = filepath.Join(projectConfig.ProjectRoot, neighborPath)
		}

		if result := checkNeighborProjectVersion(neighborName, absPath, currentVersion); !result.IsCompatible {
			incompatiblePackages = append(incompatiblePackages, result)
		}
	}

	return incompatiblePackages
}

// checkNeighborProjectVersion checks version requirement for a single neighbor project
func checkNeighborProjectVersion(neighborName, neighborPath, currentVersion string) VersionCheckResult {
	result := VersionCheckResult{
		PackageName:  neighborName,
		PackagePath:  neighborPath,
		PackageType:  "neighbor project",
		IsCompatible: true,
	}

	// Try to load the neighbor's config
	neighborConfig, err := config.LoadProjectConfig(neighborPath)
	if err != nil {
		result.IsCompatible = false
		result.Error = fmt.Errorf("failed to load neighbor project config: %w", err)
		return result
	}

	result.PackageName = neighborConfig.Name
	result.RequiredVersion = neighborConfig.Compiler.Version

	if neighborConfig.Compiler.Version != "" {
		if err := checkCompilerVersion(neighborConfig.Compiler.Version); err != nil {
			result.IsCompatible = false
			result.Error = err
		}
	}

	return result
}

func printRequired(pkg VersionCheckResult) {
	colors.RED.Printf("   • %s requires v%s\n", pkg.PackageName, pkg.RequiredVersion)
}

// formatVersionCompatibilityError formats a comprehensive error message for version incompatibilities
func formatVersionCompatibilityError(currentVersion string, incompatiblePackages []VersionCheckResult) error {
	// Print the header with colors
	colors.RED.Printf("❌ Compiler Version Mismatch\n")
	colors.YELLOW.Printf("   Current version: %s\n\n", currentVersion)

	// Group packages by type for better organization
	projectPackages := []VersionCheckResult{}
	remotePackages := []VersionCheckResult{}
	neighborPackages := []VersionCheckResult{}

	for _, pkg := range incompatiblePackages {
		switch pkg.PackageType {
		case "current project":
			projectPackages = append(projectPackages, pkg)
		case "remote dependency":
			remotePackages = append(remotePackages, pkg)
		case "neighbor project":
			neighborPackages = append(neighborPackages, pkg)
		}
	}

	// Display incompatible packages by category
	if len(projectPackages) > 0 {
		colors.BLUE.Printf("📦 Current Project:\n")
		for _, pkg := range projectPackages {
			printRequired(pkg)
		}
		fmt.Println()
	}

	if len(remotePackages) > 0 {
		colors.BLUE.Printf("🌐 Remote Dependencies:\n")
		for _, pkg := range remotePackages {
			printRequired(pkg)
		}
		fmt.Println()
	}

	if len(neighborPackages) > 0 {
		colors.BLUE.Printf("🏠 Neighbor Projects:\n")
		for _, pkg := range neighborPackages {
			printRequired(pkg)
		}
		fmt.Println()
	}

	// Provide helpful suggestions
	colors.CYAN.Printf("💡 Solutions:\n")
	colors.WHITE.Printf("   1. Update your compiler to a newer version\n")
	colors.WHITE.Printf("   2. Or adjust version requirements in package configs\n")

	// Return a simple error for the exit mechanism
	return fmt.Errorf("version compatibility check failed")
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

	// Check compiler version compatibility for all packages
	if err := checkAllPackageVersions(config); err != nil {
		// Error message is already formatted with colors in formatVersionCompatibilityError
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
