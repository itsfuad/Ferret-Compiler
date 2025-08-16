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
	err := dm.migrateFerRetVersions()
	if err != nil {
		return fmt.Errorf("failed to migrate fer.ret versions: %w", err)
	}

	dependencies := dm.configfile.Dependencies.Modules

	if len(dependencies) == 0 {
		colors.YELLOW.Println("⚠️ No dependencies found in fer.ret. Skipping installation.")
		return nil
	}

	colors.BLUE.Printf("📦 Installing %d dependencies from fer.ret...\n", len(dependencies))

	for packagename, version := range dependencies {
		if err := dm.installDependency(BuildModuleSpec(packagename, version)); err != nil {
			colors.RED.Printf("❌ Failed to install %s: %v\n", packagename, err)
		}
	}

	colors.GREEN.Println("✅ All dependencies installed successfully.")

	return dm.Save()
}

func (dm *DependencyManager) InstallDependency(packagename string) error {
	// Implementation for installing a specific dependency
	err := dm.installDependency(packagename)
	if err != nil {
		colors.RED.Printf("❌ Failed to install %s: %v\n", packagename, err)
		return err
	}

	return dm.Save()
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

func (dm *DependencyManager) installDependency(packagename string) error {
	// Implementation for installing a single dependency
	colors.BLUE.Printf("📦 Installing %s...\n", packagename)

	host, user, repo, version, err := SplitRepo(packagename)
	if err != nil {
		return err
	}

	// check what versions are available
	actualVersion, err := CheckRemoteModuleExists(host, user, repo, version)
	if err != nil {
		colors.RED.Printf("Module not found: %s\n", packagename)
		os.Exit(1)
	}

	isInstalled, err := dm.downloadIfNotCached(host, user, repo, actualVersion, packagename)
	if err != nil {
		return err
	}

	if !isInstalled {
		dm.lockfile.SetDependency(host, user, repo, actualVersion, true, []string{}, []string{})
		colors.GREEN.Printf("✅ Successfully installed %s\n", packagename)
	}

	colors.BLUE.Printf("Module %s/%s/%s@%s is already cached\n", host, user, repo, version)
	
	return nil
}

// ensureModuleCached ensures the module is cached, downloading if necessary
func (dm *DependencyManager) downloadIfNotCached(host, user, repo, version, packagename string) (bool, error) {
	if !IsModuleCached(filepath.Join(dm.projectRoot, dm.configfile.Cache.Path), filepath.Join(host, user, repo), version) {
		err := DownloadRemoteModule(host, user, repo, version, filepath.Join(dm.projectRoot, dm.configfile.Cache.Path))
		if err != nil {
			colors.RED.Printf("Failed to download module: %s\n", packagename)
			return false, err
		}
	}
	return true, nil
}

// migrateFerRetVersions is the internal migration function that doesn't trigger reinstall
func (dm *DependencyManager) migrateFerRetVersions() error {
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