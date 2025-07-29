package modules

import (
	"errors"
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

	// Add to fer.ret as direct dependency
	fullRepoPath := "github.com/" + repoName
	err = WriteFerRetDependency(dm.projectRoot, fullRepoPath, actualVersion, description)
	if err != nil {
		return fmt.Errorf("failed to update fer.ret: %w", err)
	}

	// After updating fer.ret, regenerate the lockfile
	return dm.InstallAllDependencies()
}

// InstallAllDependencies installs all dependencies listed in fer.ret and generates the lockfile
func (dm *DependencyManager) InstallAllDependencies() error {
	dependencies, err := ReadFerRetDependencies(dm.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to read dependencies from fer.ret: %w", err)
	}

	if len(dependencies) == 0 {
		colors.YELLOW.Println("No dependencies found in fer.ret")
		dm.lockfile = NewLockfile()
		dm.saveLockfile()
		return nil
	}

	colors.BLUE.Printf("Installing %d dependencies from fer.ret...\n", len(dependencies))

	lockfile := NewLockfile()
	seen := make(map[string]struct{}) // repo@version keys

	for moduleName, dep := range dependencies {
		moduleSpec := moduleName
		if dep.Version != "" {
			moduleSpec = moduleName + "@" + dep.Version
		}
		_, requestedVersion, repoName, err := SplitRemotePath(moduleSpec)
		if err != nil {
			colors.RED.Printf("Invalid module specification: %s\n", moduleSpec)
			continue
		}
		actualVersion, err := CheckRemoteModuleExists(repoName, requestedVersion)
		if err != nil {
			colors.RED.Printf("Module not found: %s\n", moduleSpec)
			continue
		}
		cachePath := filepath.Join(dm.projectRoot, ".ferret", "modules")
		if !IsModuleCached(cachePath, repoName, actualVersion) {
			err = DownloadRemoteModule(dm.projectRoot, repoName, actualVersion, cachePath)
			if err != nil {
				colors.RED.Printf("Failed to download module: %s\n", moduleSpec)
				continue
			}
		}
		fullRepoPath := "github.com/" + repoName
		key := fullRepoPath + "@" + actualVersion
		// Recursively resolve transitive dependencies and collect their keys
		transitiveKeys := dm.resolveTransitiveDependencies(fullRepoPath, actualVersion, repoName, cachePath, lockfile, seen, key)
		lockfile.SetDependency(fullRepoPath, actualVersion, true, dep.Comment, transitiveKeys, []string{})
		for _, depKey := range transitiveKeys {
			lockfile.AddUsedBy(depKey, key)
		}
		seen[key] = struct{}{}
	}

	dm.lockfile = lockfile
	return dm.saveLockfile()
}

// RemoveDependency removes a direct dependency and cleans up unused indirect dependencies
func (dm *DependencyManager) RemoveDependency(moduleName string) error {
	lockfile, err := LoadLockfile(dm.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Only allow removal of direct dependencies
	var foundAny bool
	var errs []string
	for key, entry := range lockfile.Dependencies {
		if strings.HasPrefix(key, moduleName+"@") {
			if !entry.Direct {
				errs = append(errs, fmt.Sprintf("cannot remove indirect dependency: %s", key))
				continue
			}
			foundAny = true
			if len(entry.UsedBy) > 0 {
				errs = append(errs, fmt.Sprintf("cannot delete %s. required by: %v", key, entry.UsedBy))
				continue
			}
			// Recursively remove dependencies if not used by others
			for _, depKey := range entry.Dependencies {
				lockfile.RemoveUsedBy(depKey, key)
				depEntry := lockfile.Dependencies[depKey]
				if !depEntry.Direct && len(depEntry.UsedBy) == 0 {
					lockfile.RemoveDependency(depEntryKeyParts(depKey))
					dm.deleteCacheForKey(depKey)
				}
			}
			lockfile.RemoveDependency(depEntryKeyParts(key))
			dm.deleteCacheForKey(key)
		}
	}
	if !foundAny {
		return fmt.Errorf("module %s is not installed as a direct dependency", moduleName)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	dm.lockfile = lockfile
	return dm.saveLockfile()
}

// CleanupUnusedDependencies removes indirect dependencies that are no longer used (UsedBy == 0)
func (dm *DependencyManager) CleanupUnusedDependencies() error {
	removed := 0
	for key, entry := range dm.lockfile.Dependencies {
		if !entry.Direct && len(entry.UsedBy) == 0 {
			delete(dm.lockfile.Dependencies, key)
			dm.deleteCacheForKey(key)
			removed++
		}
	}
	if removed == 0 {
		colors.GREEN.Println("No unused dependencies found")
		return nil
	}
	colors.BLUE.Printf("Cleaned up %d unused dependencies...\n", removed)
	return dm.saveLockfile()
}

// ListDependencies shows all dependencies with their status
func (dm *DependencyManager) ListDependencies() error {
	colors.BLUE.Println("Dependencies:")
	colors.BLUE.Println("============")

	// Show direct dependencies
	colors.GREEN.Println("Direct dependencies:")
	directCount := 0
	for dep, info := range dm.lockfile.Dependencies {
		if info.Direct {
			colors.GREEN.Printf("  %s@%s", dep, info.Version)
			if info.Description != "" {
				colors.GREEN.Printf(" (%s)", info.Description)
			}
			colors.GREEN.Println()
			directCount++
		}
	}
	if directCount == 0 {
		colors.GREEN.Println("  (none)")
	}

	// Show indirect dependencies
	colors.YELLOW.Println("\nIndirect dependencies:")
	indirectCount := 0
	for dep, info := range dm.lockfile.Dependencies {
		if !info.Direct {
			colors.YELLOW.Printf("  %s@%s\n", dep, info.Version)
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

// resolveTransitiveDependencies recursively resolves and adds transitive dependencies to the lockfile
// Returns the list of repo@version keys that the parent depends on
func (dm *DependencyManager) resolveTransitiveDependencies(parentRepo, parentVersion, parentRepoName, cachePath string, lockfile *Lockfile, seen map[string]struct{}, parentKey string) []string {
	moduleDir := filepath.Join(cachePath, "github.com", parentRepoName+"@"+parentVersion)
	ferFiles, err := findFerretFiles(moduleDir)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to scan module files for %s@%s: %s\n", parentRepo, parentVersion, err)
		return nil
	}
	remoteImports := make(map[string]struct{})
	for _, ferFile := range ferFiles {
		imports, err := extractRemoteImports(ferFile)
		if err != nil {
			colors.YELLOW.Printf("Warning: Failed to parse %s: %s\n", ferFile, err)
			continue
		}
		for _, imp := range imports {
			remoteImports[imp] = struct{}{}
		}
	}
	var depKeys []string
	for importPath := range remoteImports {
		_, requestedVersion, repoName, err := SplitRemotePath(importPath)
		if err != nil {
			continue
		}
		actualVersion, err := CheckRemoteModuleExists(repoName, requestedVersion)
		if err != nil {
			continue
		}
		if !IsModuleCached(cachePath, repoName, actualVersion) {
			_ = DownloadRemoteModule(dm.projectRoot, repoName, actualVersion, cachePath)
		}
		fullRepoPath := "github.com/" + repoName
		key := fullRepoPath + "@" + actualVersion
		if _, already := seen[key]; !already {
			// Recursively resolve further dependencies
			transitiveKeys := dm.resolveTransitiveDependencies(fullRepoPath, actualVersion, repoName, cachePath, lockfile, seen, key)
			lockfile.SetDependency(fullRepoPath, actualVersion, false, "", transitiveKeys, []string{parentKey})
			seen[key] = struct{}{}
		} else {
			lockfile.AddUsedBy(key, parentKey)
		}
		depKeys = append(depKeys, key)
	}
	return depKeys
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
	dm.lockfile.Dependencies[fullRepoPath+"@"+actualVersion] = LockfileEntry{
		Version:     actualVersion,
		Direct:      false,
		Description: "",
	}

	return nil
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

// Helper to split depKey into repo, version
func depEntryKeyParts(depKey string) (string, string) {
	at := strings.LastIndex(depKey, "@")
	if at == -1 {
		return depKey, ""
	}
	return depKey[:at], depKey[at+1:]
}

// deleteCacheForKey deletes the cache directory for a given repo@version key
func (dm *DependencyManager) deleteCacheForKey(depKey string) {
	repo, version := depEntryKeyParts(depKey)
	if repo == "" || version == "" {
		return
	}
	// repo is github.com/user/repo
	parts := strings.SplitN(repo, "/", 3)
	if len(parts) < 3 {
		return
	}
	repoName := parts[1] + "/" + parts[2] // user/repo
	cachePath := filepath.Join(dm.projectRoot, ".ferret", "modules", "github.com", repoName+"@"+version)
	_ = os.RemoveAll(cachePath)
}
