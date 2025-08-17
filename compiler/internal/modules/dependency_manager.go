package modules

import (
	"compiler/colors"
	"compiler/config"
	"fmt"
	"os"
	"path/filepath"
)

// DependencyManager handles dependency management using the lockfile system
type DependencyManager struct {
	projectRoot string
	lockfile    *Lockfile
	configfile  *config.ProjectConfig
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

	directDependencies := dm.configfile.Dependencies.Modules

	if len(directDependencies) == 0 {
		colors.YELLOW.Println("⚠️  No dependencies found in config file. Skipping installation...")
		return nil
	}

	colors.BLUE.Printf("📦 Installing %d dependencies from fer.ret...\n", len(directDependencies))

	for packagename, version := range directDependencies {
		if err := installDependency(dm, BuildModuleSpec(packagename, version), true); err != nil {
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

	// If a dependency has no used by references, it can be safely removed
	// Must present in fer.ret
	version, ok := dm.configfile.Dependencies.Modules[packageName]
	if !ok {
		colors.YELLOW.Printf("⚠️  Dependency %s is not listed in current project config file\n", packageName)
		return nil
	}

	key := BuildModuleSpec(packageName, version)

	entry, exists := dm.lockfile.Dependencies[key]
	if !exists {
		return fmt.Errorf("⚠️  dependency %s is not listed in lockfile", packageName)
	}

	// must be direct
	if !entry.Direct {
		return fmt.Errorf("⚠️  dependency %s is not a direct dependency", packageName)
	}

	if len(entry.UsedBy) > 0 {
		return fmt.Errorf("⚠️  dependency %s is still in use by other modules and cannot be removed", packageName)
	}

	delete(dm.configfile.Dependencies.Modules, packageName)

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
	}

	colors.BLUE.Printf("Module %s/%s/%s@%s is already cached\n", host, user, repo, version)

	if isDirect {
		// add to config file
		dm.configfile.Dependencies.Modules[fmt.Sprintf("%s/%s/%s", host, user, repo)] = actualVersion
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
	repoPath := filepath.Join(dm.projectRoot, dm.configfile.Cache.Path, host, user, BuildModuleSpec(repo, version))
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
			for dep, version := range config.Dependencies.Modules {
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

	dependencies := dm.configfile.Dependencies.Modules
	var updated bool

	for moduleName, version := range dependencies {
		normalizedVersion := NormalizeVersion(version)
		if normalizedVersion != version {
			colors.BLUE.Printf("📝 Migrating %s: %s -> %s\n", moduleName, version, normalizedVersion)
			dm.configfile.Dependencies.Modules[moduleName] = normalizedVersion
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
