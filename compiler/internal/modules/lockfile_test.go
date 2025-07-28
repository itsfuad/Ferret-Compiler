package modules

import (
	"os"
	"testing"
	"time"
)

func TestNewLockfile(t *testing.T) {
	lockfile := NewLockfile()

	if lockfile.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", lockfile.Version)
	}

	if len(lockfile.DirectDeps) != 0 {
		t.Errorf("Expected empty direct deps, got %d", len(lockfile.DirectDeps))
	}

	if len(lockfile.Dependencies) != 0 {
		t.Errorf("Expected empty dependencies, got %d", len(lockfile.Dependencies))
	}
}

func TestAddDirectDependency(t *testing.T) {
	lockfile := NewLockfile()

	// Add a direct dependency
	lockfile.AddDirectDependency("github.com/user/repo", "v1.0.0", "test dependency")

	// Check direct deps list
	if len(lockfile.DirectDeps) != 1 {
		t.Errorf("Expected 1 direct dep, got %d", len(lockfile.DirectDeps))
	}

	if lockfile.DirectDeps[0] != "github.com/user/repo" {
		t.Errorf("Expected github.com/user/repo, got %s", lockfile.DirectDeps[0])
	}

	// Check dependencies map
	entry, exists := lockfile.Dependencies["github.com/user/repo"]
	if !exists {
		t.Error("Dependency not found in map")
	}

	if entry.Version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", entry.Version)
	}

	if !entry.Direct {
		t.Error("Expected direct dependency to be marked as direct")
	}

	if entry.Description != "test dependency" {
		t.Errorf("Expected description 'test dependency', got %s", entry.Description)
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
	lockfile.AddDirectDependency("github.com/user/repo", "v1.0.0", "test dependency")
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

	if len(loadedLockfile.DirectDeps) != len(lockfile.DirectDeps) {
		t.Errorf("Direct deps count mismatch: expected %d, got %d", len(lockfile.DirectDeps), len(loadedLockfile.DirectDeps))
	}

	if len(loadedLockfile.Dependencies) != len(lockfile.Dependencies) {
		t.Errorf("Dependencies count mismatch: expected %d, got %d", len(lockfile.Dependencies), len(loadedLockfile.Dependencies))
	}
}
