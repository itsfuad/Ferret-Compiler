package modules

import (
	"compiler/constants"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type LockfileEntry struct {
	Version      string   `json:"version"`
	Direct       bool     `json:"direct"`
	Dependencies []string `json:"dependencies"`
	UsedBy       []string `json:"used_by"`
}

type Lockfile struct {
	projectRoot  string
	Version      string                   `json:"version"`
	Dependencies map[string]LockfileEntry `json:"dependencies"`
	GeneratedAt  string                   `json:"generated_at"`
}

func LoadLockfile(projectRoot string) (*Lockfile, error) {
	lockfilePath := filepath.Join(projectRoot, constants.LOCKFILE)
	lockfile := &Lockfile{
		projectRoot:  projectRoot,
		Dependencies: make(map[string]LockfileEntry),
	}

	// if doesn't exist, just return
	if _, err := os.Stat(lockfilePath); os.IsNotExist(err) {
		return lockfile, nil
	}

	data, err := os.ReadFile(lockfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	if err := json.Unmarshal(data, lockfile); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	return lockfile, nil
}

func (l *Lockfile) Save() error {
	lockfilePath := filepath.Join(l.projectRoot, constants.LOCKFILE)
	l.Version = constants.LOCKFILE_VERSION
	l.GeneratedAt = time.Now().Format(time.RFC3339)

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile: %w", err)
	}
	if err := os.WriteFile(lockfilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}
	return nil
}

// SetNewDependency adds or updates a dependency in the lockfile
func (l *Lockfile) SetNewDependency(host, user, repo, version string, isDirect bool) {

	key := fmt.Sprintf("%s/%s/%s@%s", host, user, repo, version)

	// If entry already exists, preserve existing relationships
	if entry, exists := l.Dependencies[key]; exists {
		entry.Version = version
		entry.Direct = isDirect || entry.Direct // Make direct if either existing or new is direct
		l.Dependencies[key] = entry
	} else {
		// Create new entry
		l.Dependencies[key] = LockfileEntry{
			Version:      version,
			Direct:       isDirect,
			Dependencies: []string{},
			UsedBy:       []string{},
		}
	}
}

func (l *Lockfile) AddIndirectDependency(parent, child string) {
	entry, exists := l.Dependencies[parent]
	if !exists {
		return
	}
	if slices.Contains(entry.Dependencies, child) {
		return // already present
	}
	entry.Dependencies = append(entry.Dependencies, child)
	l.Dependencies[parent] = entry // Update the parent entry
	l.AddUsedBy(parent, child)
}

// AddUsedBy adds a parent to the UsedBy list for a dependency
func (l *Lockfile) AddUsedBy(parentKey, depKey string) {
	entry, exists := l.Dependencies[depKey]
	if !exists {
		return
	}
	if slices.Contains(entry.UsedBy, parentKey) {
		return // already present
	}
	entry.UsedBy = append(entry.UsedBy, parentKey)
	l.Dependencies[depKey] = entry // Update the dependency entry
}

// RemoveUsedBy removes a parent from the UsedBy list for a dependency
func (l *Lockfile) RemoveUsedBy(depKey, parentKey string) {
	entry, exists := l.Dependencies[depKey]
	if !exists {
		return
	}
	newUsedBy := make([]string, 0, len(entry.UsedBy))
	for _, u := range entry.UsedBy {
		if u != parentKey {
			newUsedBy = append(newUsedBy, u)
		}
	}
	entry.UsedBy = newUsedBy
	l.Dependencies[depKey] = entry
}

func (l *Lockfile) RemoveDependency(key string) {
	delete(l.Dependencies, key)
}

// SetDirect sets the direct flag for a dependency
func (l *Lockfile) SetDirect(key string, isDirect bool) {
	entry, exists := l.Dependencies[key]
	if !exists {
		return
	}
	entry.Direct = isDirect
	l.Dependencies[key] = entry
}

func (l *Lockfile) GetDependency(repo, version string) (LockfileEntry, bool) {
	key := BuildPackageSpec(repo, version)
	entry, exists := l.Dependencies[key]
	return entry, exists
}

func (l *Lockfile) GetDependencyVersion(repo, version string) (string, bool) {
	key := BuildPackageSpec(repo, version)
	entry, exists := l.Dependencies[key]
	if !exists {
		return "", false
	}
	return entry.Version, true
}

func (l *Lockfile) GetAllDependencies() map[string]LockfileEntry {
	result := make(map[string]LockfileEntry)
	maps.Copy(result, l.Dependencies)
	return result
}
