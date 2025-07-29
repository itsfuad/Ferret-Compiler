package modules

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewLockfile(t *testing.T) {
	lockfile := NewLockfile()

	if lockfile.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", lockfile.Version)
	}

	if len(lockfile.Dependencies) != 0 {
		t.Errorf("Expected empty dependencies, got %d", len(lockfile.Dependencies))
	}
}

func TestSaveAndLoadLockfile(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "lockfile-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a lockfile
	lockfile := NewLockfile()
	lockfile.SetDependency("github.com/user/repo", "v1.0.0", true, "test dependency", []string{}, []string{})
	lockfile.GeneratedAt = time.Now().Format(time.RFC3339)

	// Save it
	err = SaveLockfile(tempDir, lockfile)
	if err != nil {
		t.Fatalf("Failed to save lockfile: %v", err)
	}

	// Load it back
	loadedLockfile, err := LoadLockfile(tempDir)
	if err != nil {
		t.Fatalf("Failed to load lockfile: %v", err)
	}

	// Check that the data is the same
	if loadedLockfile.Version != lockfile.Version {
		t.Errorf("Version mismatch: expected %s, got %s", lockfile.Version, loadedLockfile.Version)
	}

	if len(loadedLockfile.Dependencies) != len(lockfile.Dependencies) {
		t.Errorf("Dependencies count mismatch: expected %d, got %d", len(lockfile.Dependencies), len(loadedLockfile.Dependencies))
	}
}

func TestSetDependencyAndUsedBy(t *testing.T) {
	lockfile := NewLockfile()
	lockfile.SetDependency("github.com/user/repo", "v1.0.0", true, "test", []string{"github.com/user/dep@v2.0.0"}, []string{})
	lockfile.SetDependency("github.com/user/dep", "v2.0.0", false, "dep", []string{}, []string{"github.com/user/repo@v1.0.0"})

	entry, exists := lockfile.Dependencies["github.com/user/repo@v1.0.0"]
	if !exists || entry.Version != "v1.0.0" || !entry.Direct {
		t.Errorf("Direct dependency not set correctly")
	}
	depEntry, exists := lockfile.Dependencies["github.com/user/dep@v2.0.0"]
	if !exists || depEntry.Direct {
		t.Errorf("Indirect dependency not set correctly")
	}
	if len(depEntry.UsedBy) != 1 || depEntry.UsedBy[0] != "github.com/user/repo@v1.0.0" {
		t.Errorf("UsedBy not set correctly")
	}
}

func TestAddRemoveUsedBy(t *testing.T) {
	lockfile := NewLockfile()
	lockfile.SetDependency("github.com/user/dep", "v2.0.0", false, "dep", []string{}, []string{})
	lockfile.AddUsedBy("github.com/user/dep@v2.0.0", "github.com/user/repo@v1.0.0")
	entry := lockfile.Dependencies["github.com/user/dep@v2.0.0"]
	if len(entry.UsedBy) != 1 {
		t.Errorf("AddUsedBy failed")
	}
	lockfile.RemoveUsedBy("github.com/user/dep@v2.0.0", "github.com/user/repo@v1.0.0")
	entry = lockfile.Dependencies["github.com/user/dep@v2.0.0"]
	if len(entry.UsedBy) != 0 {
		t.Errorf("RemoveUsedBy failed")
	}
}

func TestRecursiveRemovalAndCacheCleanup(t *testing.T) {
	lockfile := NewLockfile()
	// Simulate a cache dir
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "github.com", "user", "dep@v2.0.0")
	os.MkdirAll(cacheDir, 0755)
	// Set up dependencies
	lockfile.SetDependency("github.com/user/repo", "v1.0.0", true, "test", []string{"github.com/user/dep@v2.0.0"}, []string{})
	lockfile.SetDependency("github.com/user/dep", "v2.0.0", false, "dep", []string{}, []string{"github.com/user/repo@v1.0.0"})
	// Remove used_by and check recursive removal
	lockfile.RemoveUsedBy("github.com/user/dep@v2.0.0", "github.com/user/repo@v1.0.0")
	depEntry := lockfile.Dependencies["github.com/user/dep@v2.0.0"]
	if len(depEntry.UsedBy) != 0 {
		t.Errorf("UsedBy not removed correctly")
	}
	// Simulate recursive removal and cache cleanup
	delete(lockfile.Dependencies, "github.com/user/dep@v2.0.0")
	os.RemoveAll(cacheDir)
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Errorf("Cache directory not cleaned up")
	}
}
