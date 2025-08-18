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

	// If a dependency has no used by references, it can be safely removed
	// Must present in fer.ret
	version, ok := dm.configfile.Dependencies.Packages[packageName]
	if !ok {
		return fmt.Errorf("⚠️  dependency %s is not listed in fer.ret", packageName)
	}

	key := BuildPackageSpec(packageName, version)

	entry, exists := dm.lockfile.Dependencies[key]
	if !exists {
		return fmt.Errorf("⚠️  dependency %s is not listed in lockfile", packageName)
	}

	// must be direct
	if !entry.Direct {
		return fmt.Errorf("⚠️  dependency %s is not a direct dependency", packageName)
	}

	if len(entry.UsedBy) > 0 {
		return fmt.Errorf("⚠️  dependency %s is still in use by other packages and cannot be removed", packageName)
	}

	delete(dm.configfile.Dependencies.Packages, packageName)

	cachesToDelete := []string{key}

	// remove it's dependencies from the lockfile
	if len(entry.Dependencies) > 0 {
		// remove its name from that deps, used by
		for _, dep := range entry.Dependencies {
			dm.lockfile.RemoveUsedBy(dep, key)
			if len(dm.lockfile.Dependencies[dep].UsedBy) == 0 {
				cachesToDelete = append(cachesToDelete, dep)
			}
		}
	}

	for _, cache := range cachesToDelete {
		dm.lockfile.RemoveDependency(cache)
		// delete cache
		cachePath := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path, cache)
		if err := os.RemoveAll(cachePath); err != nil {
			return fmt.Errorf("❌ failed to delete cache for %s: %v", cache, err)
		}
	}

	dm.Save()
	dm.cleanupEmptyDirectories()

	return nil
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

			installDependency(dm, update.Name, true)
			// remive old version
			key := fmt.Sprintf("%s/%s/%s@%s", host, user, repo, update.CurrentVersion)
			dm.lockfile.RemoveDependency(key)
			colors.GREEN.Printf("✅ Updated %s to version %s\n", update.Name, update.LatestVersion)
		}
	}

	// save
	dm.Save()

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

	for pkg := range cachedPackages {
		if _, found := dm.lockfile.Dependencies[pkg]; !found {
			orphanedPackages[pkg] = true
		}
	}

	return orphanedPackages
}

func (dm *DependencyManager) RemoveOrphanedPackages() error {
	orphanedPackages := dm.GetOrphans()
	for depKey := range orphanedPackages {
		if err := os.RemoveAll(filepath.Join(dm.projectRoot, dm.configfile.Cache.Path, depKey)); err != nil {
			return fmt.Errorf("❌ Failed to remove orphaned package %s: %w", depKey, err)
		}
		colors.BLUE.Printf("🗑️  Removed orphaned package: %s\n", depKey)
	}
	return nil
}

func (dm *DependencyManager) GetPackagesInCache() (map[string]bool, error) {
	packageDir := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path)

	if _, err := os.Stat(packageDir); err != nil {
		fmt.Printf("❌ Cache directory %s does not exist: %v\n", packageDir, err)
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

	isInstalled, err := downloadIfNotCached(dm, host, user, repo, actualVersion)
	if err != nil {
		return err
	}

	if !isInstalled {
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
		// self reference will cause infinite loop.
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
				indirectDependencies = append(indirectDependencies, fmt.Sprintf("%s@%s", dep, version))
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
