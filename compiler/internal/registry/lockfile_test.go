package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test constants to avoid string literal duplication
const (
	testTempDirPrefix  = "ferret-test"
	lockFileName       = "ferret.lock"
	expectedNoErrorMsg = "Expected no error, got %v"
	expectedVersionMsg = "Expected version 1.0, got %s"
	testTimestamp      = "2023-01-01T00:00:00Z"
	testRepoName       = "test/repo"
	testRepo1Name      = "test/repo1"
	testRepo2Name      = "test/repo2"
	testVersionV100    = "v1.0.0"
	testVersionV200    = "v2.0.0"
	testTarURL         = "https://example.com/test.tar.gz"
	testTar1URL        = "https://example.com/test1.tar.gz"
	testTar2URL        = "https://example.com/test2.tar.gz"
	testChecksumABC    = "abc123"
	testChecksumDEF    = "def456"
)

func TestLoadLockFileNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	lockFile, err := LoadLockFile(tempDir)
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	if lockFile.Version != "1.0" {
		t.Errorf(expectedVersionMsg, lockFile.Version)
	}

	if lockFile.Packages == nil {
		t.Error("Expected packages map to be initialized")
	}

	if len(lockFile.Packages) != 0 {
		t.Errorf("Expected empty packages map, got %d entries", len(lockFile.Packages))
	}
}

func TestLoadLockFileExisting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test lock file
	lockData := map[string]interface{}{
		"version":      "1.0",
		"generated_at": testTimestamp,
		"packages": map[string]interface{}{
			testRepoName: map[string]interface{}{
				"version":       testVersionV100,
				"resolved_url":  testTarURL,
				"checksum":      testChecksumABC,
				"downloaded_at": testTimestamp,
			},
		},
	}

	data, err := json.Marshal(lockData)
	if err != nil {
		t.Fatal(err)
	}

	lockPath := filepath.Join(tempDir, lockFileName)
	err = os.WriteFile(lockPath, data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	lockFile, err := LoadLockFile(tempDir)
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	if lockFile.Version != "1.0" {
		t.Errorf(expectedVersionMsg, lockFile.Version)
	}

	if len(lockFile.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(lockFile.Packages))
	}

	entry, exists := lockFile.Packages[testRepoName]
	if !exists {
		t.Error("Expected test/repo package to exist")
	}

	if entry.Version != testVersionV100 {
		t.Errorf("Expected version v1.0.0, got %s", entry.Version)
	}
}

func TestLoadLockFileInvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	lockPath := filepath.Join(tempDir, lockFileName)
	err = os.WriteFile(lockPath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadLockFile(tempDir)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestSaveLockFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	lockFile := &LockFile{
		Version:  "1.0",
		Packages: make(map[string]LockEntry),
	}

	lockFile.Packages[testRepoName] = LockEntry{
		Version:     testVersionV100,
		ResolvedURL: testTarURL,
		Checksum:    testChecksumABC,
		Downloaded:  testTimestamp,
	}

	err = SaveLockFile(tempDir, lockFile)
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	// Verify file was created
	lockPath := filepath.Join(tempDir, lockFileName)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}

	// Verify content
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}

	var savedLock LockFile
	err = json.Unmarshal(data, &savedLock)
	if err != nil {
		t.Fatal(err)
	}

	if savedLock.Version != "1.0" {
		t.Errorf(expectedVersionMsg, savedLock.Version)
	}

	if len(savedLock.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(savedLock.Packages))
	}
}

func TestGetLockedVersion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a lock file with a package
	lockFile := &LockFile{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Packages:    make(map[string]LockEntry),
	}

	lockFile.Packages[testRepoName] = LockEntry{
		Version:     testVersionV100,
		ResolvedURL: testTarURL,
		Checksum:    testChecksumABC,
		Downloaded:  time.Now().UTC().Format(time.RFC3339),
	}

	err = SaveLockFile(tempDir, lockFile)
	if err != nil {
		t.Fatal(err)
	}

	// Test existing package
	version, err := GetLockedVersion(tempDir, testRepoName)
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	if version != testVersionV100 {
		t.Errorf("Expected version v1.0.0, got %s", version)
	}

	// Test non-existing package
	version, err = GetLockedVersion(tempDir, "nonexistent/repo")
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	if version != "" {
		t.Errorf("Expected empty version, got %s", version)
	}
}

func TestRemoveModuleFromLockfileNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	lockPath := filepath.Join(tempDir, lockFileName)

	err = RemoveModuleFromLockfile(lockPath, testRepoName)
	if err != nil {
		t.Errorf("Expected no error for non-existent lockfile, got %v", err)
	}
}

func TestRemoveModuleFromLockfileExisting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a lock file with packages
	lockData := map[string]interface{}{
		"version":      "1.0",
		"generated_at": testTimestamp,
		"packages": map[string]interface{}{
			testRepo1Name: map[string]interface{}{
				"version":       testVersionV100,
				"resolved_url":  testTar1URL,
				"checksum":      testChecksumABC,
				"downloaded_at": testTimestamp,
			},
			testRepo2Name: map[string]interface{}{
				"version":       testVersionV200,
				"resolved_url":  testTar2URL,
				"checksum":      testChecksumDEF,
				"downloaded_at": testTimestamp,
			},
		},
	}

	data, err := json.Marshal(lockData)
	if err != nil {
		t.Fatal(err)
	}

	lockPath := filepath.Join(tempDir, lockFileName)
	err = os.WriteFile(lockPath, data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Remove one module
	err = RemoveModuleFromLockfile(lockPath, testRepo1Name)
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	// Verify the module was removed
	content, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}

	var lockfile map[string]interface{}
	err = json.Unmarshal(content, &lockfile)
	if err != nil {
		t.Fatal(err)
	}

	packages := lockfile["packages"].(map[string]interface{})
	if _, exists := packages[testRepo1Name]; exists {
		t.Error("Expected test/repo1 to be removed")
	}

	if _, exists := packages[testRepo2Name]; !exists {
		t.Error("Expected test/repo2 to remain")
	}
}

func TestRemoveModuleFromLockfileInvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	lockPath := filepath.Join(tempDir, lockFileName)
	err = os.WriteFile(lockPath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = RemoveModuleFromLockfile(lockPath, testRepoName)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestCalculateDirectoryChecksum(t *testing.T) {
	tempDir, err := os.MkdirTemp("", testTempDirPrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFile1 := filepath.Join(tempDir, "file1.txt")
	err = os.WriteFile(testFile1, []byte("test content 1"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	testFile2 := filepath.Join(tempDir, "file2.txt")
	err = os.WriteFile(testFile2, []byte("test content 2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	checksum, err := calculateDirectoryChecksum(tempDir)
	if err != nil {
		t.Fatalf(expectedNoErrorMsg, err)
	}

	if checksum == "" {
		t.Error("Expected non-empty checksum")
	}

	// Test with non-existent directory
	_, err = calculateDirectoryChecksum(filepath.Join(tempDir, "nonexistent"))
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}
