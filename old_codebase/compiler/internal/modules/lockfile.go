package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const LockfileName = "ferret.lock"

type LockfileEntry struct {
	Version      string   `json:"version"`
	Direct       bool     `json:"direct"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	UsedBy       []string `json:"used_by"`
}

type Lockfile struct {
	Version      string                   `json:"version"`
	Dependencies map[string]LockfileEntry `json:"dependencies"`
	GeneratedAt  string                   `json:"generated_at"`
}

func NewLockfile() *Lockfile {
	return &Lockfile{
		Version:      "1.0",
		Dependencies: make(map[string]LockfileEntry),
	}
}

func LoadLockfile(projectRoot string) (*Lockfile, error) {
	lockfilePath := filepath.Join(projectRoot, LockfileName)
	data, err := os.ReadFile(lockfilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewLockfile(), nil
		}
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}
	var lockfile Lockfile
	if err := json.Unmarshal(data, &lockfile); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}
	return &lockfile, nil
}

func SaveLockfile(projectRoot string, lockfile *Lockfile) error {
	lockfilePath := filepath.Join(projectRoot, LockfileName)
	sortedDeps := make([]string, 0, len(lockfile.Dependencies))
	for dep := range lockfile.Dependencies {
		sortedDeps = append(sortedDeps, dep)
	}
	sort.Strings(sortedDeps)
	outputLockfile := &Lockfile{
		Version:      lockfile.Version,
		Dependencies: make(map[string]LockfileEntry),
		GeneratedAt:  lockfile.GeneratedAt,
	}
	for _, dep := range sortedDeps {
		outputLockfile.Dependencies[dep] = lockfile.Dependencies[dep]
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
	key := repo + "@" + version
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
	for _, u := range entry.UsedBy {
		if u == parentKey {
			return // already present
		}
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
	key := repo + "@" + version
	delete(l.Dependencies, key)
}

func (l *Lockfile) GetDependency(repo, version string) (LockfileEntry, bool) {
	key := repo + "@" + version
	entry, exists := l.Dependencies[key]
	return entry, exists
}

func (l *Lockfile) GetDependencyVersion(repo, version string) (string, bool) {
	key := repo + "@" + version
	entry, exists := l.Dependencies[key]
	if !exists {
		return "", false
	}
	return entry.Version, true
}

func (l *Lockfile) GetAllDependencies() map[string]LockfileEntry {
	result := make(map[string]LockfileEntry)
	for k, v := range l.Dependencies {
		result[k] = v
	}
	return result
}

func (l *Lockfile) ValidateLockfile() error {
	return nil
}
