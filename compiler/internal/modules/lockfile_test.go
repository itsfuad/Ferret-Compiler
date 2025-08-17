package modules

import (
	"compiler/constants"
	"reflect"
	"testing"
)

const V1 = "v1.0.0"

// Helper to create a temp project root
func tempProjectRoot(t *testing.T) string {
	dir := t.TempDir()
	return dir
}

func TestLoadLockfileFileNotExist(t *testing.T) {
	root := tempProjectRoot(t)
	lockfile, err := LoadLockfile(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lockfile == nil {
		t.Fatal("expected lockfile, got nil")
	}
	if len(lockfile.Dependencies) != 0 {
		t.Errorf("expected empty dependencies, got %v", lockfile.Dependencies)
	}
}

func TestLockfileSaveAndLoad(t *testing.T) {
	root := tempProjectRoot(t)
	lockfile := &Lockfile{
		projectRoot: root,
		Version:     "",
		Dependencies: map[string]LockfileEntry{
			"host/user/repo@v1.2.3": {
				Version:      "v1.2.3",
				Direct:       true,
				Dependencies: []string{"host/user/dep@v0.1.0"},
				UsedBy:       []string{},
			},
		},
		GeneratedAt: "",
	}
	err := lockfile.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadLockfile(root)
	if err != nil {
		t.Fatalf("LoadLockfile failed: %v", err)
	}
	if loaded.Version != constants.LOCKFILE_VERSION {
		t.Errorf("expected version %s, got %s", constants.LOCKFILE_VERSION, loaded.Version)
	}
	if len(loaded.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(loaded.Dependencies))
	}
	if loaded.GeneratedAt == "" {
		t.Error("expected GeneratedAt to be set")
	}
}

func TestLockfileSetNewDependency(t *testing.T) {
	lockfile := &Lockfile{
		Dependencies: make(map[string]LockfileEntry),
	}
	lockfile.SetNewDependency("host", "user", "repo", V1, true)
	key := "host/user/repo@v1.0.0"
	entry, ok := lockfile.Dependencies[key]
	if !ok {
		t.Fatalf("dependency not set")
	}
	if entry.Version != V1 || !entry.Direct {
		t.Errorf("unexpected entry: %+v", entry)
	}
}

func TestLockfileAddIndirectDependency(t *testing.T) {
	lockfile := &Lockfile{
		Dependencies: make(map[string]LockfileEntry),
	}
	parent := "host/user/parent@v1.0.0"
	child := "host/user/child@v2.0.0"
	lockfile.Dependencies[parent] = LockfileEntry{}
	lockfile.Dependencies[child] = LockfileEntry{}
	lockfile.AddIndirectDependency(parent, child)
	if !contains(lockfile.Dependencies[parent].Dependencies, child) {
		t.Errorf("child not added to parent's dependencies")
	}
	if !contains(lockfile.Dependencies[child].UsedBy, parent) {
		t.Errorf("parent not added to child's UsedBy")
	}
}

func TestLockfileAddUsedBy(t *testing.T) {
	lockfile := &Lockfile{
		Dependencies: map[string]LockfileEntry{
			"dep": {},
		},
	}
	lockfile.AddUsedBy("parent", "dep")
	if !contains(lockfile.Dependencies["dep"].UsedBy, "parent") {
		t.Error("parent not added to UsedBy")
	}
	// Should not duplicate
	lockfile.AddUsedBy("parent", "dep")
	if count := countOccurrences(lockfile.Dependencies["dep"].UsedBy, "parent"); count != 1 {
		t.Error("parent duplicated in UsedBy")
	}
}

func TestLockfileRemoveUsedBy(t *testing.T) {
	lockfile := &Lockfile{
		Dependencies: map[string]LockfileEntry{
			"dep": {UsedBy: []string{"parent", "other"}},
		},
	}
	lockfile.RemoveUsedBy("dep", "parent")
	if contains(lockfile.Dependencies["dep"].UsedBy, "parent") {
		t.Error("parent not removed from UsedBy")
	}
	if !contains(lockfile.Dependencies["dep"].UsedBy, "other") {
		t.Error("other should remain in UsedBy")
	}
}

func TestLockfileRemoveDependency(t *testing.T) {
	lockfile := &Lockfile{
		Dependencies: map[string]LockfileEntry{
			"dep": {},
		},
	}
	lockfile.RemoveDependency("dep")
	if _, ok := lockfile.Dependencies["dep"]; ok {
		t.Error("dependency not removed")
	}
}

func TestLockfileGetDependency(t *testing.T) {
	lockfile := &Lockfile{
		Dependencies: map[string]LockfileEntry{
			"repo@v1.0.0": {Version: V1},
		},
	}
	entry, ok := lockfile.GetDependency("repo", V1)
	if !ok || entry.Version != V1 {
		t.Error("GetDependency failed")
	}
}

func TestLockfileGetDependencyVersion(t *testing.T) {
	lockfile := &Lockfile{
		Dependencies: map[string]LockfileEntry{
			"repo@v1.0.0": {Version: V1},
		},
	}
	ver, ok := lockfile.GetDependencyVersion("repo", V1)
	if !ok || ver != V1 {
		t.Error("GetDependencyVersion failed")
	}
}

func TestLockfileGetAllDependencies(t *testing.T) {
	lockfile := &Lockfile{
		Dependencies: map[string]LockfileEntry{
			"a": {Version: "1"},
			"b": {Version: "2"},
		},
	}
	all := lockfile.GetAllDependencies()
	if !reflect.DeepEqual(all, lockfile.Dependencies) {
		t.Error("GetAllDependencies did not return a copy of dependencies")
	}
	all["a"] = LockfileEntry{Version: "changed"}
	if lockfile.Dependencies["a"].Version == "changed" {
		t.Error("modifying result should not affect original")
	}
}

// --- Helpers ---

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func countOccurrences(slice []string, s string) int {
	count := 0
	for _, v := range slice {
		if v == s {
			count++
		}
	}
	return count
}
