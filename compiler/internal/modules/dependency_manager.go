package modules

import (
	"compiler/colors"
	"compiler/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DependencyManager handles dependency management using the lockfile system
type DependencyManager struct {
	projectRoot string
	lockfile    *Lockfile
	configfile  *config.ProjectConfig
}

type ModuleUpdateInfo struct {
	Name           string
	CurrentVersion string
	LatestVersion  string
}

// NewDependencyManager creates a new dependency manager for the given project
func NewDependencyManager(projectRoot string) (*DependencyManager, error) {

	configfile, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to load config file: %w", err)
	}

	lockfile, err := LoadLockfile(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to load lockfile: %w", err)
	}

	return &DependencyManager{
		projectRoot: projectRoot,
		lockfile:    lockfile,
		configfile:  configfile,
	}, nil
}

// InstallAllDependencies is the internal implementation with migration control
func (dm *DependencyManager) InstallAllDependencies() error {
	// First, migrate fer.ret versions to normalized format if needed
	err := migrateFerRetVersions(dm)
	if err != nil {
		return fmt.Errorf("failed to migrate fer.ret versions: %w", err)
	}

	directDependencies := dm.configfile.Dependencies.Packages

	if len(directDependencies) == 0 {
		colors.YELLOW.Println("⚠️  No dependencies found in config file. Skipping installation...")
		return nil
	}

	colors.BLUE.Printf("📦 Installing %d dependencies from fer.ret...\n", len(directDependencies))

	for packagename, version := range directDependencies {
		if err := installDependency(dm, BuildPackageSpec(packagename, version), true); err != nil {
			colors.RED.Printf("❌ Failed to install %s: %v\n", packagename, err)
		}
	}

	colors.GREEN.Println("✅ All dependencies installed successfully.")

	return dm.Save()
}

func (dm *DependencyManager) InstallDependency(packagename string) error {
	// Implementation for installing a specific dependency
	err := installDependency(dm, packagename, true)
	if err != nil {
		return err
	}

	return dm.Save()
}

func (dm *DependencyManager) RemoveDependency(packageName string) error {
	if packageName == "" {
		return fmt.Errorf("⚠️  package name is required")
	}

	// Find package in fer.ret and lockfile
	packageInfo, err := dm.findPackageForRemoval(packageName)
	if err != nil {
		return err
	}

	// Remove from fer.ret
	delete(dm.configfile.Dependencies.Packages, packageName)

	// Handle the removal based on usage
	if len(packageInfo.lockfileEntry.UsedBy) > 0 {
		return dm.convertToIndirect(packageName, packageInfo.lockfileKey)
	}

	return dm.completelyRemovePackage(packageName, packageInfo)
}

// packageRemovalInfo holds information about a package being removed
type packageRemovalInfo struct {
	lockfileKey   string
	lockfileEntry LockfileEntry
}

// findPackageForRemoval locates package in fer.ret and lockfile, returns error for invalid states
func (dm *DependencyManager) findPackageForRemoval(packageName string) (*packageRemovalInfo, error) {
	version, inFerRet := dm.configfile.Dependencies.Packages[packageName]

	var lockfileKey string
	var lockfileEntry LockfileEntry
	var inLockfile bool

	if inFerRet {
		lockfileKey = BuildPackageSpec(packageName, version)
		lockfileEntry, inLockfile = dm.lockfile.Dependencies[lockfileKey]
	} else {
		lockfileKey, lockfileEntry, inLockfile = dm.findPackageInLockfile(packageName)
	}

	return dm.validatePackageRemovalState(packageName, inFerRet, inLockfile, lockfileKey, lockfileEntry)
}

// findPackageInLockfile searches for package in lockfile with any version
func (dm *DependencyManager) findPackageInLockfile(packageName string) (string, LockfileEntry, bool) {
	for key, entry := range dm.lockfile.Dependencies {
		if strings.HasPrefix(key, packageName+"@") {
			return key, entry, true
		}
	}
	return "", LockfileEntry{}, false
}

// validatePackageRemovalState checks package state and returns appropriate errors
func (dm *DependencyManager) validatePackageRemovalState(packageName string, inFerRet, inLockfile bool, lockfileKey string, lockfileEntry LockfileEntry) (*packageRemovalInfo, error) {
	if !inFerRet && !inLockfile {
		return nil, fmt.Errorf("📦 Package %s is not installed", packageName)
	}

	if !inFerRet && inLockfile {
		return nil, fmt.Errorf("📦 Package %s is not a direct dependency", packageName)
	}

	if inFerRet && !inLockfile {
		delete(dm.configfile.Dependencies.Packages, packageName)
		dm.Save()
		return nil, fmt.Errorf("⚠️  Package %s was in fer.ret but not in lockfile, removed from fer.ret", packageName)
	}

	return &packageRemovalInfo{
		lockfileKey:   lockfileKey,
		lockfileEntry: lockfileEntry,
	}, nil
}

// convertToIndirect marks package as indirect instead of removing it
func (dm *DependencyManager) convertToIndirect(packageName, lockfileKey string) error {
	dm.lockfile.SetDirect(lockfileKey, false)
	dm.Save()
	colors.GREEN.Printf("✅ Successfully removed %s as direct dependency (kept as indirect)\n", packageName)
	return nil
}

// completelyRemovePackage removes package and its unused transitive dependencies
func (dm *DependencyManager) completelyRemovePackage(packageName string, info *packageRemovalInfo) error {
	cachesToDelete := dm.findCachesToDelete(info)

	colors.GREEN.Printf("🗑️  Removing package %s\n", packageName)

	if err := dm.deleteCachesAndShowMessages(packageName, info.lockfileKey, cachesToDelete); err != nil {
		return err
	}

	dm.Save()
	dm.cleanupEmptyDirectories()
	return nil
}

// findCachesToDelete identifies all caches that should be deleted
func (dm *DependencyManager) findCachesToDelete(info *packageRemovalInfo) []string {
	cachesToDelete := []string{info.lockfileKey}

	for _, dep := range info.lockfileEntry.Dependencies {
		dm.lockfile.RemoveUsedBy(dep, info.lockfileKey)
		if len(dm.lockfile.Dependencies[dep].UsedBy) == 0 {
			cachesToDelete = append(cachesToDelete, dep)
		}
	}

	return cachesToDelete
}

// deleteCachesAndShowMessages removes caches and displays appropriate messages
func (dm *DependencyManager) deleteCachesAndShowMessages(packageName, lockfileKey string, cachesToDelete []string) error {
	for _, cache := range cachesToDelete {
		dm.lockfile.RemoveDependency(cache)

		if err := dm.deleteCacheDirectory(cache); err != nil {
			return err
		}

		dm.showRemovalMessage(packageName, lockfileKey, cache)
	}
	return nil
}

// deleteCacheDirectory removes the physical cache directory
func (dm *DependencyManager) deleteCacheDirectory(cache string) error {
	cachePath := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path, cache)
	if err := os.RemoveAll(cachePath); err != nil {
		return fmt.Errorf("❌ failed to delete cache for %s: %v", cache, err)
	}
	return nil
}

// showRemovalMessage displays appropriate message for removed package
func (dm *DependencyManager) showRemovalMessage(packageName, lockfileKey, cache string) {
	if cache == lockfileKey {
		colors.GREEN.Printf("✅ Successfully removed %s\n", packageName)
	} else {
		parts := strings.Split(cache, "@")
		if len(parts) > 0 {
			colors.BLUE.Printf("🗑️  Also removed unused transitive dependency: %s\n", parts[0])
		}
	}
}

func (dm *DependencyManager) Save() error {
	// save lockfile
	if err := dm.lockfile.Save(); err != nil {
		colors.RED.Printf("❌ Failed to save lockfile: %v\n", err)
		return err
	}

	// save config
	if err := dm.configfile.Save(); err != nil {
		colors.RED.Printf("❌ Failed to save config file: %v\n", err)
		return err
	}

	return nil
}

func (dm *DependencyManager) cleanupEmptyDirectories() {
	// walk all dir and clean up empty ones
	err := filepath.Walk(filepath.Join(dm.projectRoot, dm.configfile.Cache.Path), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// check if directory is empty
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				colors.YELLOW.Printf("🗑️  Removing empty directory: %s\n", path)
				return os.RemoveAll(path)
			}
		}
		return nil
	})
	if err != nil {
		colors.RED.Printf("❌ Failed to clean up empty directories: %v\n", err)
	}
}

func (dm *DependencyManager) CheckForAvailableUpdates(packagename string) []ModuleUpdateInfo {

	var updates []ModuleUpdateInfo

	colors.BLUE.Println("🔍 Sniffing for module updates...")

	if packagename == "" {
		updates = checkAllUpdates(dm)
	} else {
		updates = checkUpdateForPackage(dm, packagename)
	}

	if len(updates) > 0 {
		colors.YELLOW.Printf("⚠️  Found updates for the following packages:\n")
		for _, update := range updates {
			colors.YELLOW.Printf("📦 %s: %s -> %s\n", update.Name, update.CurrentVersion, update.LatestVersion)
		}
	} else {
		colors.GREEN.Println("✅ All up to date")
	}

	return updates
}

func checkUpdateForPackage(dm *DependencyManager, packagename string) []ModuleUpdateInfo {

	var updates []ModuleUpdateInfo
	// get the installed version from the config file
	version, ok := dm.configfile.Dependencies.Packages[packagename]
	if !ok {
		colors.RED.Printf("❌ Package %q is not installed\n", packagename)
		return nil
	}

	host, user, repo, _, err := SplitRepo(packagename)
	if err != nil {
		colors.RED.Printf("❌ Failed to parse package name %q: %v\n", packagename, err)
		return nil
	}

	_, latestVersion, err := CheckAndGetActualVersion(host, user, repo, "")
	if err != nil {
		colors.RED.Printf("❌ Failed to check for updates for %q: %v\n", packagename, err)
	}

	if hasUpdate(version, latestVersion) {
		// check if has cache
		updates = append(updates, ModuleUpdateInfo{
			Name:           packagename,
			CurrentVersion: version,
			LatestVersion:  latestVersion,
		})
	}
	return updates
}

func checkAllUpdates(dm *DependencyManager) []ModuleUpdateInfo {
	var updates []ModuleUpdateInfo
	// check if any direct dependency has a newer version available
	for dep, version := range dm.configfile.Dependencies.Packages {
		key := BuildPackageSpec(dep, version)
		host, user, repo, _, err := SplitRepo(key)
		if err != nil {
			colors.RED.Printf("❌ Failed to parse dependency %s: %v\n", dep, err)
			continue
		}

		_, latestVersion, err := CheckAndGetActualVersion(host, user, repo, "")
		if err != nil {
			colors.RED.Printf("❌ Failed to check for updates for %s: %v\n", dep, err)
			continue
		}

		if hasUpdate(version, latestVersion) {
			// check if has cache
			updates = append(updates, ModuleUpdateInfo{
				Name:           dep,
				CurrentVersion: version,
				LatestVersion:  latestVersion,
			})
		}
	}
	return updates
}

func (dm *DependencyManager) AutoUpdate(packagename string) error {
	updates := dm.CheckForAvailableUpdates(packagename)
	if len(updates) > 0 {
		// start the update process
		for _, update := range updates {
			host, user, repo, _, err := SplitRepo(update.Name)
			if err != nil {
				colors.RED.Printf("❌ Failed to parse package name %s: %v\n", update.Name, err)
				continue
			}

			// Remove old version first to clean up old dependencies
			oldKey := fmt.Sprintf("%s/%s/%s@%s", host, user, repo, update.CurrentVersion)
			if err := dm.removeOldVersionForUpdate(oldKey); err != nil {
				colors.RED.Printf("❌ Failed to remove old version %s: %v\n", oldKey, err)
				continue
			}

			// Install new version with proper version specification
			newPackageSpec := fmt.Sprintf("%s@%s", update.Name, update.LatestVersion)
			if err := installDependency(dm, newPackageSpec, true); err != nil {
				colors.RED.Printf("❌ Failed to update %s: %v\n", update.Name, err)
				continue
			}

			colors.GREEN.Printf("✅ Updated %s to version %s\n", update.Name, update.LatestVersion)
		}
	}

	// save
	dm.Save()

	return nil
}

// removeOldVersionForUpdate removes an old version during update, cleaning up unused transitive dependencies
func (dm *DependencyManager) removeOldVersionForUpdate(oldKey string) error {
	entry, exists := dm.lockfile.Dependencies[oldKey]
	if !exists {
		return nil // already removed
	}

	// Clean up transitive dependencies that might become unused
	for _, dep := range entry.Dependencies {
		dm.lockfile.RemoveUsedBy(dep, oldKey)
		// If the dependency is no longer used by anyone and is not direct, remove it
		if depEntry, exists := dm.lockfile.Dependencies[dep]; exists {
			if !depEntry.Direct && len(depEntry.UsedBy) == 0 {
				dm.lockfile.RemoveDependency(dep)
				// Also remove its cache directory
				cachePath := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path, dep)
				os.RemoveAll(cachePath)
				colors.BLUE.Printf("🗑️  Removed unused transitive dependency: %s\n", dep)
			}
		}
	}

	// Remove the old version itself
	dm.lockfile.RemoveDependency(oldKey)

	// Remove its cache directory
	cachePath := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path, oldKey)
	if err := os.RemoveAll(cachePath); err != nil {
		return fmt.Errorf("failed to remove cache for %s: %w", oldKey, err)
	}

	return nil
}

func (dm *DependencyManager) GetOrphans() map[string]bool {
	// cached but not listed in lockfile are called orphans
	cachedPackages, err := dm.GetPackagesInCache()
	if err != nil {
		colors.RED.Printf("❌ Failed to get cached packages: %v\n", err)
		return nil
	}

	orphanedPackages := make(map[string]bool)

	// Find cached packages not in lockfile
	for pkg := range cachedPackages {
		if _, found := dm.lockfile.Dependencies[pkg]; !found {
			orphanedPackages[pkg] = true
		}
	}

	// Find lockfile entries that are not used by anything and not direct dependencies
	for depKey, entry := range dm.lockfile.Dependencies {
		if !entry.Direct && len(entry.UsedBy) == 0 {
			orphanedPackages[depKey] = true
		}
	}

	return orphanedPackages
}

func (dm *DependencyManager) RemoveOrphanedPackages() error {
	orphanedPackages := dm.GetOrphans()
	for depKey := range orphanedPackages {
		// Remove from cache if it exists
		cachePath := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path, depKey)
		if _, err := os.Stat(cachePath); err == nil {
			if err := os.RemoveAll(cachePath); err != nil {
				return fmt.Errorf("❌ Failed to remove orphaned package cache %s: %w", depKey, err)
			}
			colors.BLUE.Printf("🗑️  Removed orphaned package cache: %s\n", depKey)
		}

		// Remove from lockfile if it exists
		if _, found := dm.lockfile.Dependencies[depKey]; found {
			dm.lockfile.RemoveDependency(depKey)
			colors.BLUE.Printf("🗑️  Removed orphaned lockfile entry: %s\n", depKey)
		}
	}

	// Save lockfile after cleanup
	if len(orphanedPackages) > 0 {
		dm.Save()
	}

	return nil
}

func (dm *DependencyManager) GetPackagesInCache() (map[string]bool, error) {
	packageDir := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path)

	if _, err := os.Stat(packageDir); err != nil {
		// No package directory means no cached modules
		return nil, nil
	}

	cachedPackages := make(map[string]bool)

	err := filepath.WalkDir(packageDir, func(path string, d os.DirEntry, err error) error {
		return handleCacheDirEntry(packageDir, path, d, err, cachedPackages)
	})

	if err != nil {
		return nil, err
	}

	return cachedPackages, nil
}

// handleCacheDirEntry processes a single directory entry for GetPackagesInCache
func handleCacheDirEntry(packageDir, path string, d os.DirEntry, err error, cachedPackages map[string]bool) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {
		return nil
	}

	rel, err := filepath.Rel(packageDir, path)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}

	parts := strings.Split(filepath.ToSlash(rel), "/")

	// we only care about depth = 3 → host/owner/repo@version
	if len(parts) == 3 {
		if _, _, _, _, err := SplitRepo(strings.Join(parts, "/")); err == nil {
			cachedPackages[strings.Join(parts, "/")] = true
		}
		// don’t go deeper inside this repo
		return filepath.SkipDir
	}

	// if depth > 3, skip immediately
	if len(parts) > 3 {
		return filepath.SkipDir
	}

	return nil
}

func installDependency(dm *DependencyManager, packagename string, isDirect bool) error {
	// Implementation for installing a single dependency

	host, user, repo, version, err := SplitRepo(packagename)
	if err != nil {
		return err
	}

	colors.BLUE.Printf("📦 Installing %s/%s/%s@%s\n", host, user, repo, version)

	// check what versions are available
	actualVersion, err := CheckRemoteModuleExists(host, user, repo, version)
	if err != nil {
		colors.RED.Printf("Package %s/%s/%s@%s not found on %s\n", host, user, repo, version, host)
		os.Exit(1)
	}

	alreadyCached, err := downloadIfNotCached(dm, host, user, repo, actualVersion)
	if err != nil {
		return err
	}

	if alreadyCached {
		colors.GREEN.Printf("✅ Successfully installed %s/%s/%s@%s\n", host, user, repo, actualVersion)
	} else {
		colors.BLUE.Printf("Module %s/%s/%s@%s is already cached\n", host, user, repo, version)
	}

	if isDirect {
		// add to config file
		dm.configfile.Dependencies.Packages[fmt.Sprintf("%s/%s/%s", host, user, repo)] = actualVersion
	}

	dm.lockfile.SetNewDependency(host, user, repo, actualVersion, isDirect)

	err = installTransitiveDependencies(dm, host, user, repo, actualVersion)
	if err != nil {
		return err
	}

	return nil
}

func installTransitiveDependencies(dm *DependencyManager, host, user, repo, version string) error {

	// read the currently installed package's config file
	repoPath := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path, host, user, BuildPackageSpec(repo, version))
	parent := fmt.Sprintf("%s/%s/%s@%s", host, user, repo, version)

	indirectDependencies, err := getTrasitiveList(repoPath)
	if err != nil {
		return err
	}

	// install each transitive dependency
	for _, pkg := range indirectDependencies {
		colors.LIGHT_GREEN.Printf("📦 Found transitive dependency: %s\n", pkg)

		// self reference will cause infinite loop and should be completely ignored
		if pkg == parent {
			colors.YELLOW.Printf("⚠️  Skipping self-referential transitive dependency: %s\n", pkg)
			continue
		}

		colors.LIGHT_GREEN.Printf("📦 Installing transitive dependency: %s\n", pkg)
		if err := installDependency(dm, pkg, false); err != nil {
			return err
		}

		// update parent lockfile AFTER the dependency is installed
		dm.lockfile.AddIndirectDependency(parent, pkg)
	}

	return nil
}

func getTrasitiveList(repoPath string) ([]string, error) {

	var indirectDependencies []string

	// walk all folders, for all fer.ret files found, install their dependencies
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// if has a fer.ret file (os.Stat)
		if _, err := os.Stat(filepath.Join(path, "fer.ret")); err == nil {
			// read this file
			config, err := config.LoadProjectConfig(path)
			if err != nil {
				return err
			}

			// install each transitive dependency
			for dep, version := range config.Dependencies.Packages {
				normalizedVersion := NormalizeVersion(version)
				indirectDependencies = append(indirectDependencies, fmt.Sprintf("%s@%s", dep, normalizedVersion))
			}
		}
		return nil
	})

	return indirectDependencies, err
}

// ensureModuleCached ensures the module is cached, downloading if necessary
func downloadIfNotCached(dm *DependencyManager, host, user, repo, version string) (bool, error) {
	if !IsModuleCached(filepath.Join(dm.projectRoot, dm.configfile.Cache.Path), filepath.Join(host, user, repo), version) {
		err := DownloadRemoteModule(host, user, repo, version, filepath.Join(dm.projectRoot, dm.configfile.Cache.Path))
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// migrateFerRetVersions is the internal migration function that doesn't trigger reinstall
func migrateFerRetVersions(dm *DependencyManager) error {
	colors.BLUE.Println("🔄 Checking fer.ret for version format migration...")

	dependencies := dm.configfile.Dependencies.Packages
	var updated bool

	for moduleName, version := range dependencies {
		normalizedVersion := NormalizeVersion(version)
		if normalizedVersion != version {
			colors.BLUE.Printf("📝 Migrating %s: %s -> %s\n", moduleName, version, normalizedVersion)
			dm.configfile.Dependencies.Packages[moduleName] = normalizedVersion
			dm.configfile.Save()
			updated = true
		}
	}

	if updated {
		colors.GREEN.Println("✅ Successfully migrated fer.ret versions to use 'v' prefix")
	} else {
		colors.GREEN.Println("✅ fer.ret versions are already normalized")
	}

	return nil
}
