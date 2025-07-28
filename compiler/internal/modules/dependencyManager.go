package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"compiler/colors"
)

// DependencyManager handles dependency management using the lockfile system
type DependencyManager struct {
	projectRoot string
	lockfile    *Lockfile
}

// NewDependencyManager creates a new dependency manager for the given project
func NewDependencyManager(projectRoot string) (*DependencyManager, error) {
	lockfile, err := LoadLockfile(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load lockfile: %w", err)
	}

	return &DependencyManager{
		projectRoot: projectRoot,
		lockfile:    lockfile,
	}, nil
}

// InstallDirectDependency installs a direct dependency and its transitive dependencies
func (dm *DependencyManager) InstallDirectDependency(moduleSpec, description string) error {
	// Parse the module specification
	_, requestedVersion, repoName, err := SplitRemotePath(moduleSpec)
	if err != nil {
		return fmt.Errorf("invalid module specification: %w", err)
	}

	colors.BLUE.Printf("Installing direct dependency: %s", moduleSpec)
	if requestedVersion != "latest" {
		colors.BLUE.Printf(" (version: %s)", requestedVersion)
	}
	colors.BLUE.Println()

	// Check if the module exists on GitHub and get the actual version
	actualVersion, err := CheckRemoteModuleExists(repoName, requestedVersion)
	if err != nil {
		return fmt.Errorf("module not found: %w", err)
	}

	colors.GREEN.Printf("Found version: %s\n", actualVersion)

	// Set up cache path
	cachePath := filepath.Join(dm.projectRoot, ".ferret", "modules")
	err = os.MkdirAll(cachePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Check if already cached
	if IsModuleCached(cachePath, repoName, actualVersion) {
		colors.YELLOW.Printf("Module %s@%s is already cached\n", repoName, actualVersion)
	} else {
		// Download and cache the module
		err = DownloadRemoteModule(dm.projectRoot, repoName, actualVersion, cachePath)
		if err != nil {
			return fmt.Errorf("failed to download module: %w", err)
		}
	}

	// Add to lockfile as direct dependency
	fullRepoPath := "github.com/" + repoName
	dm.lockfile.AddDirectDependency(fullRepoPath, actualVersion, description)

	// Install transitive dependencies
	err = dm.installTransitiveDependencies(fullRepoPath, repoName, actualVersion, cachePath)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to install transitive dependencies: %s\n", err)
		// Don't fail the entire installation for transitive dependency issues
	}

	// Save the updated lockfile
	return dm.saveLockfile()
}

// InstallAllDependencies installs all dependencies listed in fer.ret
func (dm *DependencyManager) InstallAllDependencies() error {
	// Read direct dependencies from fer.ret
	dependencies, err := ReadFerRetDependencies(dm.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to read dependencies from fer.ret: %w", err)
	}

	if len(dependencies) == 0 {
		colors.YELLOW.Println("No dependencies found in fer.ret")
		return nil
	}

	colors.BLUE.Printf("Installing %d dependencies from fer.ret...\n", len(dependencies))

	// Install each direct dependency
	for moduleName, dep := range dependencies {
		moduleSpec := moduleName
		if dep.Version != "" {
			moduleSpec = moduleName + "@" + dep.Version
		}

		err := dm.InstallDirectDependency(moduleSpec, dep.Comment)
		if err != nil {
			colors.RED.Printf("Failed to install %s: %s\n", moduleName, err)
			// Continue with other dependencies
		}
	}

	return nil
}

// RemoveDependency removes a direct dependency and cleans up unused indirect dependencies
func (dm *DependencyManager) RemoveDependency(moduleName string) error {
	// Check if it's a direct dependency
	if !dm.lockfile.IsDirectDependency(moduleName) {
		return fmt.Errorf("module %s is not a direct dependency", moduleName)
	}

	colors.BLUE.Printf("Removing dependency: %s\n", moduleName)

	// Get all dependencies that are used by this module
	dependenciesToCheck := dm.getDependenciesUsedBy(moduleName)

	// Remove the direct dependency
	dm.lockfile.RemoveDependency(moduleName)

	// Remove usage tracking for all dependencies used by this module
	for _, dep := range dependenciesToCheck {
		dm.lockfile.RemoveDependencyUsage(dep, moduleName)
	}

	// Get unused dependencies
	unused := dm.lockfile.GetUnusedDependencies()
	if len(unused) > 0 {
		colors.YELLOW.Printf("Found %d unused dependencies that can be cleaned up:\n", len(unused))
		for _, dep := range unused {
			colors.YELLOW.Printf("  - %s\n", dep)
		}
	}

	// Save the updated lockfile
	return dm.saveLockfile()
}

// CleanupUnusedDependencies removes dependencies that are no longer used
func (dm *DependencyManager) CleanupUnusedDependencies() error {
	unused := dm.lockfile.GetUnusedDependencies()
	if len(unused) == 0 {
		colors.GREEN.Println("No unused dependencies found")
		return nil
	}

	colors.BLUE.Printf("Cleaning up %d unused dependencies...\n", len(unused))

	for _, dep := range unused {
		colors.YELLOW.Printf("Removing unused dependency: %s\n", dep)
		dm.lockfile.RemoveDependency(dep)
	}

	return dm.saveLockfile()
}

// ListDependencies shows all dependencies with their status
func (dm *DependencyManager) ListDependencies() error {
	colors.BLUE.Println("Dependencies:")
	colors.BLUE.Println("============")

	// Show direct dependencies
	colors.GREEN.Println("Direct dependencies:")
	for _, dep := range dm.lockfile.GetDirectDependencies() {
		info, _ := dm.lockfile.GetDependencyInfo(dep)
		colors.GREEN.Printf("  %s@%s", dep, info.Version)
		if info.Description != "" {
			colors.GREEN.Printf(" (%s)", info.Description)
		}
		colors.GREEN.Println()
	}

	// Show indirect dependencies
	colors.YELLOW.Println("\nIndirect dependencies:")
	indirectCount := 0
	for moduleName, info := range dm.lockfile.GetAllDependencies() {
		if !info.Direct {
			colors.YELLOW.Printf("  %s@%s (used by: %s)\n", moduleName, info.Version, strings.Join(info.UsedBy, ", "))
			indirectCount++
		}
	}

	if indirectCount == 0 {
		colors.YELLOW.Println("  (none)")
	}

	return nil
}

// findFerretFiles recursively finds all .fer files in a directory
func findFerretFiles(dir string) ([]string, error) {
	var ferFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".fer") {
			ferFiles = append(ferFiles, path)
		}

		return nil
	})

	return ferFiles, err
}

// installTransitiveDependencies finds and installs dependencies of a downloaded module
func (dm *DependencyManager) installTransitiveDependencies(parentModule, parentRepoName, parentVersion, cachePath string) error {
	moduleDir := filepath.Join(cachePath, "github.com", parentRepoName+"@"+parentVersion)

	// Find all .fer files in the module
	ferFiles, err := findFerretFiles(moduleDir)
	if err != nil {
		return fmt.Errorf("failed to scan module files: %w", err)
	}

	// Extract remote imports from all .fer files
	remoteImports := make(map[string]bool) // Use map to avoid duplicates

	for _, ferFile := range ferFiles {
		imports, err := extractRemoteImports(ferFile)
		if err != nil {
			colors.YELLOW.Printf("Warning: Failed to parse %s: %s\n", ferFile, err)
			continue
		}

		for _, imp := range imports {
			remoteImports[imp] = true
		}
	}

	// Install each unique remote dependency
	for importPath := range remoteImports {
		err := dm.installTransitiveDependency(importPath, parentModule, cachePath)
		if err != nil {
			colors.YELLOW.Printf("Warning: Failed to install transitive dependency %s: %s\n", importPath, err)
			// Continue with other dependencies
		}
	}

	return nil
}

// extractRemoteImports parses a .fer file and extracts remote import statements
func extractRemoteImports(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var remoteImports []string
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for import statements: import "github.com/..."
		if strings.HasPrefix(line, "import") && strings.Contains(line, "github.com/") {
			// Extract the import path from quotes
			start := strings.Index(line, `"`)
			if start == -1 {
				continue
			}
			end := strings.Index(line[start+1:], `"`)
			if end == -1 {
				continue
			}

			importPath := line[start+1 : start+1+end]

			// Check if it's a remote import
			if strings.HasPrefix(importPath, "github.com/") {
				remoteImports = append(remoteImports, importPath)
			}
		}
	}

	return remoteImports, nil
}

// installTransitiveDependency installs a single transitive dependency
func (dm *DependencyManager) installTransitiveDependency(importPath, parentModule, cachePath string) error {
	// Parse the remote import to get repo information
	_, requestedVersion, repoName, err := SplitRemotePath(importPath)
	if err != nil {
		return fmt.Errorf("invalid remote import path: %w", err)
	}

	// Check if the module exists on GitHub and get the actual version
	actualVersion, err := CheckRemoteModuleExists(repoName, requestedVersion)
	if err != nil {
		return fmt.Errorf("module not found: %w", err)
	}

	// Check if already cached
	if IsModuleCached(cachePath, repoName, actualVersion) {
		colors.YELLOW.Printf("Transitive dependency %s@%s is already cached\n", repoName, actualVersion)
	} else {
		// Download and cache the module
		err = DownloadRemoteModule(dm.projectRoot, repoName, actualVersion, cachePath)
		if err != nil {
			return fmt.Errorf("failed to download transitive dependency: %w", err)
		}
	}

	// Add to lockfile as indirect dependency
	fullRepoPath := "github.com/" + repoName
	dm.lockfile.AddIndirectDependency(fullRepoPath, actualVersion, parentModule)

	return nil
}

// getDependenciesUsedBy returns all dependencies that are used by the given module
func (dm *DependencyManager) getDependenciesUsedBy(moduleName string) []string {
	var dependencies []string

	for depName, info := range dm.lockfile.GetAllDependencies() {
		for _, user := range info.UsedBy {
			if user == moduleName {
				dependencies = append(dependencies, depName)
				break
			}
		}
	}

	return dependencies
}

// saveLockfile saves the lockfile with a timestamp
func (dm *DependencyManager) saveLockfile() error {
	dm.lockfile.GeneratedAt = time.Now().Format(time.RFC3339)
	return SaveLockfile(dm.projectRoot, dm.lockfile)
}

// GetLockfile returns the current lockfile
func (dm *DependencyManager) GetLockfile() *Lockfile {
	return dm.lockfile
}
