package modules

import (
	"compiler/config"
	"compiler/constants"
	"compiler/internal/testutil"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const expectedNoError = "Expected no error, got %v"

// Test module constants to avoid duplicate string literals
const (
	testMod        = "github.com/user/test-mod"
	testModVersion = "v1.2.3"
	testModWithVer = "github.com/user/test-mod@v1.2.3"

	depMod        = "github.com/user/dep-mod"
	depModVersion = "v0.5.0"
	depModWithVer = "github.com/user/dep-mod@v0.5.0"

	selfRefMod     = "github.com/user/self-ref-mod"
	selfRefVersion = "v2.0.0"
	selfRefWithVer = "github.com/user/self-ref-mod@v2.0.0"

	transitiveMod     = "github.com/user/transitive-mod"
	transitiveVersion = "v1.0.0"
	transitiveWithVer = "github.com/user/transitive-mod@v1.0.0"

	updatedMod     = "github.com/user/updated-mod"
	updatedVersion = "v2.1.0"
)

// Mock data for testing
var mockRemoteVersions = map[string]string{
	testMod:       testModVersion,
	depMod:        depModVersion,
	selfRefMod:    selfRefVersion,
	transitiveMod: transitiveVersion,
	updatedMod:    updatedVersion,
}

var mockModuleDependencies = map[string]map[string]string{
	testModWithVer: {
		depMod: depModVersion,
	},
	depModWithVer: {
		transitiveMod: transitiveVersion,
	},
	selfRefWithVer: {
		selfRefMod: selfRefVersion, // self reference
	},
	transitiveWithVer: {},
}

// Create a testable dependency manager that uses mocked remote functions
func createTestDM(t *testing.T, projectRoot string) *testDependencyManager {
	dm, err := NewDependencyManager(projectRoot)
	if err != nil {
		t.Fatalf("Failed to create dependency manager: %v", err)
	}

	return &testDependencyManager{
		DependencyManager: dm,
		t:                 t,
	}
}

// testDependencyManager wraps DependencyManager with mocked remote functions
type testDependencyManager struct {
	*DependencyManager
	t *testing.T
}

// Override installDependency with mocked version
func (tdm *testDependencyManager) installDependency(packagename string, isDirect bool) error {
	host, user, repo, version, err := SplitRepo(packagename)
	if err != nil {
		return err
	}

	// Mock remote check
	actualVersion, err := tdm.mockCheckRemoteModuleExists(host, user, repo, version)
	if err != nil {
		return fmt.Errorf("package %s/%s/%s@%s not found: %v", host, user, repo, version, err)
	}

	// Mock download
	err = tdm.mockDownloadRemoteModule(host, user, repo, actualVersion)
	if err != nil {
		return err
	}

	if isDirect {
		// add to config file
		tdm.configfile.Dependencies.Packages[fmt.Sprintf("%s/%s/%s", host, user, repo)] = actualVersion
	}

	tdm.lockfile.SetNewDependency(host, user, repo, actualVersion, isDirect)

	err = tdm.installTransitiveDependencies(host, user, repo, actualVersion)
	if err != nil {
		return err
	}

	return nil
}

func (tdm *testDependencyManager) installTransitiveDependencies(host, user, repo, version string) error {
	// Create mock directory for reading dependencies
	repoPath := filepath.Join(tdm.projectRoot, tdm.configfile.Cache.Path, host, user, BuildPackageSpec(repo, version))
	parent := fmt.Sprintf("%s/%s/%s@%s", host, user, repo, version)

	indirectDependencies, err := tdm.getTrasitiveList(repoPath)
	if err != nil {
		return err
	}

	// install each transitive dependency
	for _, pkg := range indirectDependencies {
		// self reference will cause infinite loop and should be completely ignored
		if pkg == parent {
			continue
		}

		if err := tdm.installDependency(pkg, false); err != nil {
			return err
		}

		// update parent lockfile AFTER the dependency is installed
		tdm.lockfile.AddIndirectDependency(parent, pkg)
	}

	return nil
}

func (tdm *testDependencyManager) getTrasitiveList(repoPath string) ([]string, error) {
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

// Mock functions
func (tdm *testDependencyManager) mockCheckRemoteModuleExists(host, user, repo, version string) (string, error) {
	key := fmt.Sprintf("%s/%s/%s", host, user, repo)
	if mockVersion, exists := mockRemoteVersions[key]; exists {
		if version == "" || version == "latest" {
			return mockVersion, nil
		}
		if version == mockVersion {
			return version, nil
		}
		return "", fmt.Errorf("version %s not found", version)
	}
	return "", fmt.Errorf("package not found")
}

func (tdm *testDependencyManager) mockDownloadRemoteModule(host, user, repo, version string) error {
	cacheDir := filepath.Join(tdm.projectRoot, tdm.configfile.Cache.Path)
	moduleDir := filepath.Join(cacheDir, host, user, fmt.Sprintf("%s@%s", repo, version))

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return err
	}

	// Create fer.ret with mock dependencies
	key := fmt.Sprintf("%s/%s/%s@%s", host, user, repo, version)
	deps := mockModuleDependencies[key]

	var configContent strings.Builder
	configContent.WriteString(fmt.Sprintf(`[default]
name = "%s"
version = "%s"

[compiler]
version = "0.0.1"

[build]
entry = ""
output = "bin"

`, repo, version))

	if len(deps) > 0 {
		configContent.WriteString("[dependencies]\n")
		for dep, ver := range deps {
			configContent.WriteString(fmt.Sprintf(`%s = "%s"`+"\n", dep, ver))
		}
		configContent.WriteString("\n")
	} else {
		// Empty dependencies section
		configContent.WriteString("[dependencies]\n\n")
	}

	configContent.WriteString(`[cache]
path = "backup"
`)

	return os.WriteFile(filepath.Join(moduleDir, constants.CONFIG_FILE), []byte(configContent.String()), 0644)
}

// Wrapper methods to use mocked versions
func (tdm *testDependencyManager) InstallAllDependencies() error {
	directDependencies := tdm.configfile.Dependencies.Packages

	if len(directDependencies) == 0 {
		return nil
	}

	for packagename, version := range directDependencies {
		if err := tdm.installDependency(BuildPackageSpec(packagename, version), true); err != nil {
			return fmt.Errorf("failed to install %s: %v", packagename, err)
		}
	}

	return tdm.Save()
}

func (tdm *testDependencyManager) InstallDependency(packagename string) error {
	err := tdm.installDependency(packagename, true)
	if err != nil {
		return err
	}
	return tdm.Save()
}

// Test setup helpers
func setupTestDependencyManager(t *testing.T) (*testDependencyManager, string) {
	projectRoot := testutil.CreateTempProject(t)

	// Create a proper fer.ret config file
	configContent := `name = "test-project"

[dependencies]

[cache]
path = "backup"
`
	if err := os.WriteFile(filepath.Join(projectRoot, constants.CONFIG_FILE), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create fer.ret file: %v", err)
	}

	tdm := createTestDM(t, projectRoot)
	return tdm, projectRoot
}

func addDependencyToConfig(t *testing.T, tdm *testDependencyManager, packageName, version string) {
	tdm.configfile.Dependencies.Packages[packageName] = version
	if err := tdm.configfile.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
}

// Test cases
func TestNewDependencyManager(t *testing.T) {
	projectRoot := testutil.CreateTempProject(t)

	dm, err := NewDependencyManager(projectRoot)
	if err != nil {
		t.Fatalf(expectedNoError, err)
	}

	if dm == nil {
		t.Fatal("Expected dependency manager, got nil")
	}

	if dm.projectRoot != projectRoot {
		t.Errorf("Expected project root %s, got %s", projectRoot, dm.projectRoot)
	}
}

func TestInstallAllDependenciesEmpty(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	err := tdm.InstallAllDependencies()
	if err != nil {
		t.Fatalf("Expected no error for empty dependencies, got %v", err)
	}
}

func TestInstallAllDependenciesSingleDependency(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	// Add a dependency to config
	addDependencyToConfig(t, tdm, testMod, testModVersion)

	err := tdm.InstallAllDependencies()
	if err != nil {
		t.Fatalf(expectedNoError, err)
	}

	// Verify lockfile has the dependency
	entry, exists := tdm.lockfile.Dependencies[testModWithVer]
	if !exists {
		t.Fatal("Expected dependency in lockfile")
	}

	if !entry.Direct {
		t.Error("Expected dependency to be marked as direct")
	}

	if entry.Version != "v1.2.3" {
		t.Errorf("Expected version v1.2.3, got %s", entry.Version)
	}
}

func TestInstallAllDependenciesWithTransitiveDependencies(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	// Add a dependency that has transitive dependencies
	addDependencyToConfig(t, tdm, testMod, testModVersion)

	err := tdm.InstallAllDependencies()
	if err != nil {
		t.Fatalf(expectedNoError, err)
	}

	// Verify direct dependency
	entry, exists := tdm.lockfile.Dependencies[testModWithVer]
	if !exists {
		t.Fatal("Expected direct dependency in lockfile")
	}
	if !entry.Direct {
		t.Error("Expected dependency to be marked as direct")
	}

	// Verify transitive dependency
	transEntry, exists := tdm.lockfile.Dependencies["github.com/user/dep-mod@v0.5.0"]
	if !exists {
		t.Fatal("Expected transitive dependency in lockfile")
	}
	if transEntry.Direct {
		t.Error("Expected transitive dependency to not be marked as direct")
	}

	// Verify used_by relationship
	if len(transEntry.UsedBy) != 1 || transEntry.UsedBy[0] != testModWithVer {
		t.Errorf("Expected transitive dependency used_by to be [%s], got %v", testModWithVer, transEntry.UsedBy)
	}

	// Verify dependencies relationship
	if len(entry.Dependencies) != 1 || entry.Dependencies[0] != depModWithVer {
		t.Errorf("Expected direct dependency to have [%s], got %v", depModWithVer, entry.Dependencies)
	}
}

func TestInstallAllDependenciesSelfReferentialDependency(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	// Add a dependency that has self-reference
	addDependencyToConfig(t, tdm, selfRefMod, selfRefVersion)

	err := tdm.InstallAllDependencies()
	if err != nil {
		t.Fatalf(expectedNoError, err)
	}

	// Verify the self-referencing dependency is installed
	entry, exists := tdm.lockfile.Dependencies[selfRefWithVer]
	if !exists {
		t.Fatal("Expected self-referencing dependency in lockfile")
	}

	// Verify it doesn't have itself in dependencies or used_by
	for _, dep := range entry.Dependencies {
		if dep == selfRefWithVer {
			t.Error("Self-referencing dependency should not appear in its own dependencies list")
		}
	}

	for _, user := range entry.UsedBy {
		if user == selfRefWithVer {
			t.Error("Self-referencing dependency should not appear in its own used_by list")
		}
	}
}

func TestInstallDependency(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	err := tdm.InstallDependency(testModWithVer)
	if err != nil {
		t.Fatalf(expectedNoError, err)
	}

	// Verify it's in both config and lockfile
	version, exists := tdm.configfile.Dependencies.Packages[testMod]
	if !exists || version != testModVersion {
		t.Error("Expected dependency to be added to config file")
	}

	entry, exists := tdm.lockfile.Dependencies[testModWithVer]
	if !exists || !entry.Direct {
		t.Error("Expected dependency to be added to lockfile as direct")
	}
}

func TestRemoveDependencyDirectDependency(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	// Install a dependency first
	addDependencyToConfig(t, tdm, testMod, testModVersion)
	tdm.InstallAllDependencies()

	// Remove it
	err := tdm.RemoveDependency(testMod)
	if err != nil {
		t.Fatalf(expectedNoError, err)
	}

	// Verify it's removed from config
	_, exists := tdm.configfile.Dependencies.Packages[testMod]
	if exists {
		t.Error("Expected dependency to be removed from config")
	}
}

func TestRemoveDependencyNonExistentPackage(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	err := tdm.RemoveDependency("github.com/user/non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent package")
	}

	expectedMsg := "Package github.com/user/non-existent is not installed"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestRemoveDependencyIndirectDependency(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	// Install a dependency with transitive deps
	addDependencyToConfig(t, tdm, testMod, testModVersion)
	tdm.InstallAllDependencies()

	// Try to remove the transitive dependency directly
	err := tdm.RemoveDependency(depMod)
	if err == nil {
		t.Fatal("Expected error when trying to remove indirect dependency")
	}

	expectedMsg := "Package github.com/user/dep-mod is not a direct dependency"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestInstallAllDependenciesWithComplexTransitiveDependencies(t *testing.T) {
	tdm, _ := setupTestDependencyManager(t)

	// Set up complex dependency chain: test-mod -> dep-mod -> transitive-mod
	addDependencyToConfig(t, tdm, testMod, testModVersion)

	err := tdm.InstallAllDependencies()
	if err != nil {
		t.Fatalf(expectedNoError, err)
	}

	// Verify all levels are installed correctly
	testModEntry, exists := tdm.lockfile.Dependencies[testModWithVer]
	if !exists {
		t.Fatal("Expected test-mod to be installed")
	}
	validateLockfileEntry(t, testModEntry, testModVersion, true)

	depModEntry, exists := tdm.lockfile.Dependencies[depModWithVer]
	if !exists {
		t.Fatal("Expected dep-mod to be installed")
	}
	validateLockfileEntry(t, depModEntry, depModVersion, false)

	transitiveModEntry, exists := tdm.lockfile.Dependencies[transitiveWithVer]
	if !exists {
		t.Fatal("Expected transitive-mod to be installed")
	}
	validateLockfileEntry(t, transitiveModEntry, transitiveVersion, false)

	// Verify dependency chains
	if len(testModEntry.Dependencies) != 1 || testModEntry.Dependencies[0] != depModWithVer {
		t.Errorf("Expected test-mod to depend on dep-mod, got %v", testModEntry.Dependencies)
	}

	if len(depModEntry.Dependencies) != 1 || depModEntry.Dependencies[0] != transitiveWithVer {
		t.Errorf("Expected dep-mod to depend on transitive-mod, got %v", depModEntry.Dependencies)
	}

	// Verify used_by relationships
	if len(depModEntry.UsedBy) != 1 || depModEntry.UsedBy[0] != testModWithVer {
		t.Errorf("Expected dep-mod to be used by test-mod, got %v", depModEntry.UsedBy)
	}

	if len(transitiveModEntry.UsedBy) != 1 || transitiveModEntry.UsedBy[0] != depModWithVer {
		t.Errorf("Expected transitive-mod to be used by dep-mod, got %v", transitiveModEntry.UsedBy)
	}
}

// Helper function to validate lockfile structure
func validateLockfileEntry(t *testing.T, entry LockfileEntry, expectedVersion string, expectedDirect bool) {
	if entry.Version != expectedVersion {
		t.Errorf("Expected version %s, got %s", expectedVersion, entry.Version)
	}
	if entry.Direct != expectedDirect {
		t.Errorf("Expected direct %v, got %v", expectedDirect, entry.Direct)
	}
}
