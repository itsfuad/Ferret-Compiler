package modules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ferret/colors"
	"ferret/internal/config"
)

const (
	CONFIG_DIR = ".ferret"
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
	cachePath := filepath.Join(dm.projectRoot, CONFIG_DIR, "modules")
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
	fullRepoPath := REMOTE_HOST + repoName
	err = WriteFerRetDependency(dm.projectRoot, fullRepoPath, actualVersion, description)
	if err != nil {
		return fmt.Errorf("failed to update fer.ret: %w", err)
	}

	// After updating fer.ret, regenerate the lockfile
	return dm.InstallAllDependencies()
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

// InstallAllDependencies installs all dependencies listed in fer.ret and generates the lockfile
func (dm *DependencyManager) InstallAllDependencies() error {
	dependencies, err := ReadFerRetDependencies(dm.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to read dependencies from fer.ret: %w", err)
	}

	if len(dependencies) == 0 {
		return dm.handleNoDependencies()
	}

	colors.BLUE.Printf("Installing %d dependencies from fer.ret...\n", len(dependencies))

	lockfile := NewLockfile()
	seen := make(map[string]struct{}) // repo@version keys

	for moduleName, dep := range dependencies {
		dm.installSingleDependency(moduleName, dep, lockfile, seen)
	}

	dm.lockfile = lockfile
	return dm.saveLockfile()
}

// handleNoDependencies handles the case when no dependencies are found
func (dm *DependencyManager) handleNoDependencies() error {
	colors.YELLOW.Println("No dependencies found in fer.ret")
	dm.lockfile = NewLockfile()
	dm.saveLockfile()
	return nil
}

// installSingleDependency installs a single dependency and its transitive dependencies
func (dm *DependencyManager) installSingleDependency(moduleName string, dep FerRetDependency, lockfile *Lockfile, seen map[string]struct{}) {
	moduleSpec := dm.buildModuleSpec(moduleName, dep.Version)

	_, requestedVersion, repoName, err := SplitRemotePath(moduleSpec)
	if err != nil {
		colors.RED.Printf("Invalid module specification: %s\n", moduleSpec)
		return
	}

	actualVersion, err := CheckRemoteModuleExists(repoName, requestedVersion)
	if err != nil {
		colors.RED.Printf("Module not found: %s\n", moduleSpec)
		return
	}

	if !dm.ensureModuleCached(repoName, actualVersion, moduleSpec) {
		return
	}

	dm.addDependencyToLockfile(repoName, actualVersion, dep, lockfile, seen)
}

// buildModuleSpec builds the module specification string
func (dm *DependencyManager) buildModuleSpec(moduleName, version string) string {
	if version != "" {
		return moduleName + "@" + version
	}
	return moduleName
}

// ensureModuleCached ensures the module is cached, downloading if necessary
func (dm *DependencyManager) ensureModuleCached(repoName, actualVersion, moduleSpec string) bool {
	cachePath := filepath.Join(dm.projectRoot, CONFIG_DIR, "modules")
	if !IsModuleCached(cachePath, repoName, actualVersion) {
		err := DownloadRemoteModule(dm.projectRoot, repoName, actualVersion, cachePath)
		if err != nil {
			colors.RED.Printf("Failed to download module: %s\n", moduleSpec)
			return false
		}
	}
	return true
}

// addDependencyToLockfile adds the dependency and its transitive dependencies to the lockfile
func (dm *DependencyManager) addDependencyToLockfile(repoName, actualVersion string, dep FerRetDependency, lockfile *Lockfile, seen map[string]struct{}) {
	fullRepoPath := REMOTE_HOST + repoName
	key := fullRepoPath + "@" + actualVersion
	cachePath := filepath.Join(dm.projectRoot, CONFIG_DIR, "modules")

	// Recursively resolve transitive dependencies and collect their keys
	transitiveKeys := dm.resolveTransitiveDependencies(fullRepoPath, actualVersion, repoName, cachePath, lockfile, seen, key)
	lockfile.SetDependency(fullRepoPath, actualVersion, true, dep.Comment, transitiveKeys, []string{})

	for _, depKey := range transitiveKeys {
		lockfile.AddUsedBy(depKey, key)
	}
	seen[key] = struct{}{}
}

// RemoveDependency removes a direct dependency and cleans up unused indirect dependencies
func (dm *DependencyManager) RemoveDependency(moduleName string) error {
	lockfile, err := LoadLockfile(dm.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	foundAny, errs := dm.processDependencyRemoval(moduleName, lockfile)

	if !foundAny {
		return fmt.Errorf("module %s is not installed as a direct dependency", moduleName)
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	dm.lockfile = lockfile
	return dm.saveLockfile()
}

// processDependencyRemoval processes the removal of dependencies matching the module name
func (dm *DependencyManager) processDependencyRemoval(moduleName string, lockfile *Lockfile) (bool, []string) {
	var foundAny bool
	var errs []string

	for key, entry := range lockfile.Dependencies {
		if strings.HasPrefix(key, moduleName+"@") {
			found, err := dm.removeSingleDependency(key, entry, lockfile)
			if found {
				foundAny = true
			}
			if err != "" {
				errs = append(errs, err)
			}
		}
	}

	return foundAny, errs
}

// removeSingleDependency removes a single dependency entry
func (dm *DependencyManager) removeSingleDependency(key string, entry LockfileEntry, lockfile *Lockfile) (bool, string) {
	if !entry.Direct {
		return false, fmt.Sprintf("cannot remove indirect dependency: %s", key)
	}

	if len(entry.UsedBy) > 0 {
		return true, fmt.Sprintf("cannot delete %s. required by: %v", key, entry.UsedBy)
	}

	dm.removeTransitiveDependencies(entry, key, lockfile)
	lockfile.RemoveDependency(depEntryKeyParts(key))
	dm.deleteCacheForKey(key)

	return true, ""
}

// removeTransitiveDependencies recursively removes transitive dependencies if not used by others
func (dm *DependencyManager) removeTransitiveDependencies(entry LockfileEntry, parentKey string, lockfile *Lockfile) {
	for _, depKey := range entry.Dependencies {
		lockfile.RemoveUsedBy(depKey, parentKey)
		depEntry := lockfile.Dependencies[depKey]
		if !depEntry.Direct && len(depEntry.UsedBy) == 0 {
			lockfile.RemoveDependency(depEntryKeyParts(depKey))
			dm.deleteCacheForKey(depKey)
		}
	}
}

// resolveTransitiveDependencies recursively resolves and adds transitive dependencies to the lockfile
// Returns the list of repo@version keys that the parent depends on
func (dm *DependencyManager) resolveTransitiveDependencies(parentRepo, parentVersion, parentRepoName, cachePath string, lockfile *Lockfile, seen map[string]struct{}, parentKey string) []string {
	remoteImports := dm.extractRemoteImportsFromModule(parentRepo, parentVersion, parentRepoName, cachePath)
	if remoteImports == nil {
		return nil
	}

	return dm.processRemoteImports(remoteImports, cachePath, lockfile, seen, parentKey)
}

// extractRemoteImportsFromModule extracts all remote imports from a module
func (dm *DependencyManager) extractRemoteImportsFromModule(parentRepo, parentVersion, parentRepoName, cachePath string) map[string]struct{} {
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

	return remoteImports
}

// processRemoteImports processes all remote imports and returns dependency keys
func (dm *DependencyManager) processRemoteImports(remoteImports map[string]struct{}, cachePath string, lockfile *Lockfile, seen map[string]struct{}, parentKey string) []string {
	var depKeys []string

	for importPath := range remoteImports {
		depKey := dm.processSingleRemoteImport(importPath, cachePath, lockfile, seen, parentKey)
		if depKey != "" {
			depKeys = append(depKeys, depKey)
		}
	}

	return depKeys
}

// processSingleRemoteImport processes a single remote import and returns its dependency key
func (dm *DependencyManager) processSingleRemoteImport(importPath, cachePath string, lockfile *Lockfile, seen map[string]struct{}, parentKey string) string {
	_, requestedVersion, repoName, err := SplitRemotePath(importPath)
	if err != nil {
		return ""
	}

	actualVersion, err := CheckRemoteModuleExists(repoName, requestedVersion)
	if err != nil {
		return ""
	}

	if !IsModuleCached(cachePath, repoName, actualVersion) {
		_ = DownloadRemoteModule(dm.projectRoot, repoName, actualVersion, cachePath)
	}

	return dm.addOrUpdateTransitiveDependency(repoName, actualVersion, cachePath, lockfile, seen, parentKey)
}

// addOrUpdateTransitiveDependency adds or updates a transitive dependency in the lockfile
func (dm *DependencyManager) addOrUpdateTransitiveDependency(repoName, actualVersion, cachePath string, lockfile *Lockfile, seen map[string]struct{}, parentKey string) string {
	fullRepoPath := REMOTE_HOST + repoName
	key := fullRepoPath + "@" + actualVersion

	if _, already := seen[key]; !already {
		// Recursively resolve further dependencies
		transitiveKeys := dm.resolveTransitiveDependencies(fullRepoPath, actualVersion, repoName, cachePath, lockfile, seen, key)
		lockfile.SetDependency(fullRepoPath, actualVersion, false, "", transitiveKeys, []string{parentKey})
		seen[key] = struct{}{}
	} else {
		lockfile.AddUsedBy(key, parentKey)
	}

	return key
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
		if strings.HasPrefix(line, "import") && strings.Contains(line, REMOTE_HOST) {
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
			if strings.HasPrefix(importPath, REMOTE_HOST) {
				remoteImports = append(remoteImports, importPath)
			}
		}
	}

	return remoteImports, nil
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
	cachePath := filepath.Join(dm.projectRoot, CONFIG_DIR, "modules", "github.com", repoName+"@"+version)
	_ = os.RemoveAll(cachePath)
}

// UpdateDependency updates a specific dependency to its latest version
func (dm *DependencyManager) UpdateDependency(moduleSpec string) error {
	// Parse the module specification
	_, _, repoName, err := SplitRemotePath(moduleSpec)
	if err != nil {
		return fmt.Errorf("invalid module specification: %w", err)
	}

	colors.BLUE.Printf("Updating dependency: %s to latest version\n", moduleSpec)

	// Get the actual latest version number instead of using "latest" tag
	latestVersion, err := CheckRemoteModuleExists(repoName, "latest")
	if err != nil {
		return fmt.Errorf("failed to get latest version for %s: %w", moduleSpec, err)
	}

	// For updates, we'll update fer.ret directly instead of removing/adding
	// This avoids unnecessary deletion of transitive dependencies that might be reused
	err = dm.updateFerRetDependency(moduleSpec, latestVersion)
	if err != nil {
		return fmt.Errorf("failed to update fer.ret: %w", err)
	}

	// Reinstall all dependencies to update the lockfile
	err = dm.InstallAllDependencies()
	if err != nil {
		return fmt.Errorf("failed to install updated dependencies: %w", err)
	}

	colors.GREEN.Printf("Successfully updated %s to latest version\n", moduleSpec)
	return nil
}

// updateFerRetDependency updates a dependency version in fer.ret file
func (dm *DependencyManager) updateFerRetDependency(moduleSpec, newVersion string) error {
	// Use the existing WriteFerRetDependency method to update the version
	return WriteFerRetDependency(dm.projectRoot, moduleSpec, newVersion, "")
}

// UpdateAllDependencies updates all dependencies to their latest versions
func (dm *DependencyManager) UpdateAllDependencies() error {
	// First, check what updates are available
	updates, err := dm.CheckAvailableUpdates()
	if err != nil {
		return fmt.Errorf("failed to check for available updates: %w", err)
	}

	// Filter to only modules that have updates
	var modulesToUpdate []ModuleUpdateInfo
	for _, update := range updates {
		if update.HasUpdate {
			modulesToUpdate = append(modulesToUpdate, update)
		}
	}

	if len(modulesToUpdate) == 0 {
		colors.YELLOW.Println("All dependencies are already up to date.")
		return nil
	}

	colors.BLUE.Printf("Found %d dependencies to update:\n", len(modulesToUpdate))
	for _, update := range modulesToUpdate {
		colors.BLUE.Printf("  %s: %s → %s\n", update.Name, update.CurrentVersion, update.LatestVersion)
	}
	colors.BLUE.Println()

	var failed []string
	updated := 0

	for _, update := range modulesToUpdate {
		colors.BLUE.Printf("Updating %s...\n", update.Name)
		err := dm.UpdateDependency(update.Name)
		if err != nil {
			colors.RED.Printf("Failed to update %s: %v\n", update.Name, err)
			failed = append(failed, update.Name)
		} else {
			updated++
		}
	}

	if len(failed) > 0 {
		colors.YELLOW.Printf("Successfully updated %d dependencies\n", updated)
		colors.RED.Printf("Failed to update %d dependencies: %v\n", len(failed), failed)
		return fmt.Errorf("some dependencies failed to update")
	}

	colors.GREEN.Printf("Successfully updated all %d dependencies to latest versions!\n", updated)
	return nil
}

// ModuleUpdateInfo represents information about a module that can be updated
type ModuleUpdateInfo struct {
	Name           string
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
	IsDirect       bool // Whether this is a direct dependency or transitive
}

// CheckAvailableUpdates checks for available updates for all dependencies
func (dm *DependencyManager) CheckAvailableUpdates() ([]ModuleUpdateInfo, error) {
	return dm.CheckAvailableUpdatesWithOptions(false) // Default: direct only
}

// CheckAvailableUpdatesIncludingTransitive checks for updates including transitive dependencies
func (dm *DependencyManager) CheckAvailableUpdatesIncludingTransitive() ([]ModuleUpdateInfo, error) {
	return dm.CheckAvailableUpdatesWithOptions(true)
}

// CheckAvailableUpdatesWithOptions checks for available updates with option to include transitive deps
func (dm *DependencyManager) CheckAvailableUpdatesWithOptions(includeTransitive bool) ([]ModuleUpdateInfo, error) {
	var updates []ModuleUpdateInfo
	var err error

	if includeTransitive {
		updates, err = getTransitive(dm)
	} else {
		updates, err = getNonTransitive(dm)
	}

	return updates, err
}

func getTransitive(dm *DependencyManager) ([]ModuleUpdateInfo, error) {
	// Check all dependencies from lockfile (direct + transitive)
	allDeps := dm.lockfile.GetAllDependencies()
	if len(allDeps) == 0 {
		return []ModuleUpdateInfo{}, nil
	}

	updates := []ModuleUpdateInfo{}

	for depKey, entry := range allDeps {
		// Parse dependency key to get module name and current version
		moduleName, currentVersion := depEntryKeyParts(depKey)
		if moduleName == "" || currentVersion == "" {
			continue
		}

		colors.BLUE.Printf("Checking for updates for %s (%s)...\n", moduleName,
			map[bool]string{true: "direct", false: "transitive"}[entry.Direct])

		updateInfo, err := dm.checkSingleModuleUpdate(moduleName, currentVersion, entry.Direct)
		if err != nil {
			colors.YELLOW.Printf("Warning: Could not check updates for %s: %v\n", moduleName, err)
			continue
		}

		updates = append(updates, updateInfo)
	}

	return updates, nil
}

func getNonTransitive(dm *DependencyManager) ([]ModuleUpdateInfo, error) {
	// Check only direct dependencies from fer.ret
	projectConfig, err := config.LoadProjectConfig(dm.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load project configuration: %w", err)
	}

	if len(projectConfig.Dependencies.Modules) == 0 {
		return []ModuleUpdateInfo{}, nil
	}

	updates := []ModuleUpdateInfo{}

	for moduleName, currentVersion := range projectConfig.Dependencies.Modules {
		colors.BLUE.Printf("Checking for updates for %s (direct)...\n", moduleName)

		updateInfo, err := dm.checkSingleModuleUpdate(moduleName, currentVersion, true)
		if err != nil {
			colors.YELLOW.Printf("Warning: Could not check updates for %s: %v\n", moduleName, err)
			continue
		}

		updates = append(updates, updateInfo)
	}

	return updates, nil
}

// checkSingleModuleUpdate checks for updates for a single module
func (dm *DependencyManager) checkSingleModuleUpdate(moduleName, currentVersion string, isDirect bool) (ModuleUpdateInfo, error) {
	// Parse the module specification to get repo info
	_, _, repoName, err := SplitRemotePath(moduleName)
	if err != nil {
		return ModuleUpdateInfo{}, fmt.Errorf("could not parse module %s: %w", moduleName, err)
	}

	// Check latest version available
	latestVersion, err := CheckRemoteModuleExists(repoName, "latest")
	if err != nil {
		return ModuleUpdateInfo{}, fmt.Errorf("could not get latest version for %s: %w", moduleName, err)
	}

	hasUpdate := currentVersion != latestVersion && currentVersion != "latest"

	return ModuleUpdateInfo{
		Name:           moduleName,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      hasUpdate,
		IsDirect:       isDirect,
	}, nil
}
