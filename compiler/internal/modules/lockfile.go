package modules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"compiler/colors"
)

// LockFile represents the ferret.lock file structure
type LockFile struct {
	Version     string               `json:"version"`
	GeneratedAt string               `json:"generated_at"`
	Packages    map[string]LockEntry `json:"packages"`
}

// LockEntry represents a single dependency entry in the lock file
type LockEntry struct {
	Version     string `json:"version"`
	ResolvedURL string `json:"resolved_url"`
	Checksum    string `json:"checksum"`
	Downloaded  string `json:"downloaded_at"`
}

// LoadLockFile loads the ferret.lock file from the project root
func LoadLockFile(projectRoot string) (*LockFile, error) {
	lockPath := filepath.Join(projectRoot, "ferret.lock")

	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		// Create new lock file if it doesn't exist
		return &LockFile{
			Version:     "1.0",
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Packages:    make(map[string]LockEntry),
		}, nil
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lockFile LockFile
	if err := json.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	if lockFile.Packages == nil {
		lockFile.Packages = make(map[string]LockEntry)
	}

	return &lockFile, nil
}

// SaveLockFile saves the lock file to the project root
func SaveLockFile(projectRoot string, lockFile *LockFile) error {
	lockPath := filepath.Join(projectRoot, "ferret.lock")

	lockFile.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(lockFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	err = os.WriteFile(lockPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	colors.GREEN.Printf("Updated ferret.lock\n")
	return nil
}

// UpdateLockEntry adds or updates a dependency in the lock file
func UpdateLockEntry(projectRoot, repoPath, version, downloadURL string) error {
	lockFile, err := LoadLockFile(projectRoot)
	if err != nil {
		return err
	}

	// Calculate checksum of the downloaded archive
	cachePath := filepath.Join(projectRoot, ".ferret", "modules", repoPath+"@"+version)
	checksum, err := calculateDirectoryChecksum(cachePath)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to calculate checksum for %s: %v\n", repoPath, err)
		checksum = "unknown"
	}

	lockFile.Packages[repoPath] = LockEntry{
		Version:     version,
		ResolvedURL: downloadURL,
		Checksum:    checksum,
		Downloaded:  time.Now().UTC().Format(time.RFC3339),
	}

	return SaveLockFile(projectRoot, lockFile)
}

// calculateDirectoryChecksum calculates a checksum for all files in a directory
func calculateDirectoryChecksum(dirPath string) (string, error) {
	hasher := sha256.New()

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(hasher, file)
		return err
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GetLockedVersion returns the locked version for a dependency, or empty string if not locked
func GetLockedVersion(projectRoot, repoPath string) (string, error) {
	lockFile, err := LoadLockFile(projectRoot)
	if err != nil {
		return "", err
	}

	if entry, exists := lockFile.Packages[repoPath]; exists {
		return entry.Version, nil
	}

	return "", nil
}

// RemoveModuleFromLockfile removes a module from the lockfile
func RemoveModuleFromLockfile(lockfilePath, module string) error {
	if _, err := os.Stat(lockfilePath); os.IsNotExist(err) {
		return nil // Lockfile doesn't exist, nothing to do
	}

	content, err := os.ReadFile(lockfilePath)
	if err != nil {
		return fmt.Errorf("failed to read lockfile: %w", err)
	}

	// Parse the JSON lockfile
	var lockfile map[string]interface{}
	err = json.Unmarshal(content, &lockfile)
	if err != nil {
		return fmt.Errorf("failed to parse lockfile JSON: %w", err)
	}

	// Get the packages section
	packages, ok := lockfile["packages"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid lockfile format: missing packages section")
	}

	// Remove the module from packages
	if _, exists := packages[module]; exists {
		delete(packages, module)
		colors.YELLOW.Printf("Removed '%s' from lockfile\n", module)
	} else {
		colors.YELLOW.Printf("Module '%s' not found in lockfile\n", module)
	}

	// Write the updated lockfile back
	updatedContent, err := json.MarshalIndent(lockfile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile JSON: %w", err)
	}

	err = os.WriteFile(lockfilePath, updatedContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	return nil
}
