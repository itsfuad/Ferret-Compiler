package modules

import (
	"compiler/constants"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

type LockfileEntry struct {
	Version      string   `json:"version"`
	Direct       bool     `json:"direct"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	UsedBy       []string `json:"used_by"`
}

type Lockfile struct {
	projectRoot string
	Version     string                   `json:"version"`
	Dependencies map[string]LockfileEntry `json:"dependencies"`
	GeneratedAt  string                   `json:"generated_at"`
}

func LoadLockfile(projectRoot string) (*Lockfile, error) {
	lockfilePath := filepath.Join(projectRoot, constants.LOCKFILE)
	lockfile := &Lockfile{
		projectRoot: projectRoot,
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
	sortedDeps := make([]string, 0, len(l.Dependencies))
	for dep := range l.Dependencies {
		sortedDeps = append(sortedDeps, dep)
	}
	sort.Strings(sortedDeps)
	outputLockfile := &Lockfile{
		Version:      l.Version,
		Dependencies: make(map[string]LockfileEntry),
		GeneratedAt:  l.GeneratedAt,
	}
	for _, dep := range sortedDeps {
		outputLockfile.Dependencies[dep] = l.Dependencies[dep]
	}
	data, err := json.MarshalIndent(outputLockfile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile: %w", err)
	}
	if err := os.WriteFile(lockfilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}
	return nil
}

// SetDependency sets or updates a dependency in the lockfile
func (l *Lockfile) SetDependency(repo, version string, direct bool, description string, dependencies, usedBy []string) {
	key := BuildModuleSpec(repo, version)
	l.Dependencies[key] = LockfileEntry{
		Version:      version,
		Direct:       direct,
		Description:  description,
		Dependencies: dependencies,
		UsedBy:       usedBy,
	}
}

// AddUsedBy adds a parent to the UsedBy list for a dependency
func (l *Lockfile) AddUsedBy(depKey, parentKey string) {
	entry, exists := l.Dependencies[depKey]
	if !exists {
		return
	}
	if slices.Contains(entry.UsedBy, parentKey) {
		return // already present
	}
	entry.UsedBy = append(entry.UsedBy, parentKey)
	l.Dependencies[depKey] = entry
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

func (l *Lockfile) RemoveDependency(repo, version string) {
	key := BuildModuleSpec(repo, version)
	delete(l.Dependencies, key)
}

func (l *Lockfile) GetDependency(repo, version string) (LockfileEntry, bool) {
	key := BuildModuleSpec(repo, version)
	entry, exists := l.Dependencies[key]
	return entry, exists
}

func (l *Lockfile) GetDependencyVersion(repo, version string) (string, bool) {
	key := BuildModuleSpec(repo, version)
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