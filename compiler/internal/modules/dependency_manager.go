package modules

import (
	"compiler/colors"
	"compiler/config"
	"fmt"
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

	for moduleName, version := range dependencies {
		if err := dm.installDependency(BuildModuleSpec(moduleName, version)); err != nil {
			colors.RED.Printf("❌ Failed to install %s: %v\n", moduleName, err)
		}
	}

	return nil
}

func (dm *DependencyManager) installDependency(dependency string) error {
	// Implementation for installing a single dependency
	colors.BLUE.Printf("📦 Installing %s...\n", dependency)

	_, user, repo, version, err := SplitRepo(dependency)
	if err != nil {
		return err
	}

	return DownloadRemoteModule(user, repo, version, dm.configfile.Cache.Path)
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