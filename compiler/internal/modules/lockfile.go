package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const LockfileName = "ferret.lock"

// LockfileEntry represents a single dependency entry in the lockfile
type LockfileEntry struct {
	Version     string   `json:"version"`     // The actual version installed
	Direct      bool     `json:"direct"`      // Whether this is a direct dependency
	UsedBy      []string `json:"used_by"`     // List of modules that depend on this
	Description string   `json:"description"` // Optional description
}

// Lockfile represents the complete lockfile structure
type Lockfile struct {
	Version      string                   `json:"version"`      // Lockfile format version
	DirectDeps   []string                 `json:"direct_deps"`  // List of direct dependencies
	Dependencies map[string]LockfileEntry `json:"dependencies"` // All dependencies (direct + indirect)
	GeneratedAt  string                   `json:"generated_at"` // Timestamp when lockfile was generated
}

// NewLockfile creates a new empty lockfile
func NewLockfile() *Lockfile {
	return &Lockfile{
		Version:      "1.0",
		DirectDeps:   make([]string, 0),
		Dependencies: make(map[string]LockfileEntry),
	}
}

// LoadLockfile loads the lockfile from the project root
func LoadLockfile(projectRoot string) (*Lockfile, error) {
	lockfilePath := filepath.Join(projectRoot, LockfileName)

	data, err := os.ReadFile(lockfilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty lockfile if it doesn't exist
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

// SaveLockfile saves the lockfile to the project root
func SaveLockfile(projectRoot string, lockfile *Lockfile) error {
	lockfilePath := filepath.Join(projectRoot, LockfileName)

	// Sort dependencies for consistent output
	sortedDeps := make([]string, 0, len(lockfile.Dependencies))
	for dep := range lockfile.Dependencies {
		sortedDeps = append(sortedDeps, dep)
	}
	sort.Strings(sortedDeps)

	// Create a sorted version for output
	outputLockfile := &Lockfile{
		Version:      lockfile.Version,
		DirectDeps:   make([]string, len(lockfile.DirectDeps)),
		Dependencies: make(map[string]LockfileEntry),
		GeneratedAt:  lockfile.GeneratedAt,
	}

	// Copy direct deps (already sorted)
	copy(outputLockfile.DirectDeps, lockfile.DirectDeps)
	sort.Strings(outputLockfile.DirectDeps)

	// Copy dependencies in sorted order
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

// AddDirectDependency adds a direct dependency to the lockfile
func (l *Lockfile) AddDirectDependency(moduleName, version, description string) {
	// Add to direct deps list if not already present
	found := false
	for _, dep := range l.DirectDeps {
		if dep == moduleName {
			found = true
			break
		}
	}
	if !found {
		l.DirectDeps = append(l.DirectDeps, moduleName)
	}

	// Add or update the dependency entry
	l.Dependencies[moduleName] = LockfileEntry{
		Version:     version,
		Direct:      true,
		UsedBy:      []string{}, // Direct deps are not used by others in this context
		Description: description,
	}
}

// AddIndirectDependency adds an indirect dependency to the lockfile
func (l *Lockfile) AddIndirectDependency(moduleName, version, usedBy string) {
	entry, exists := l.Dependencies[moduleName]
	if !exists {
		// New indirect dependency
		entry = LockfileEntry{
			Version: version,
			Direct:  false,
			UsedBy:  []string{usedBy},
		}
	} else {
		// Existing dependency, add to used_by if not already there
		found := false
		for _, user := range entry.UsedBy {
			if user == usedBy {
				found = true
				break
			}
		}
		if !found {
			entry.UsedBy = append(entry.UsedBy, usedBy)
		}
		// Update version if different (shouldn't happen in practice)
		if entry.Version != version {
			entry.Version = version
		}
	}

	l.Dependencies[moduleName] = entry
}

// RemoveDependency removes a dependency and its usage tracking
func (l *Lockfile) RemoveDependency(moduleName string) {
	// Remove from direct deps
	for i, dep := range l.DirectDeps {
		if dep == moduleName {
			l.DirectDeps = append(l.DirectDeps[:i], l.DirectDeps[i+1:]...)
			break
		}
	}

	// Remove from dependencies map
	delete(l.Dependencies, moduleName)
}

// RemoveDependencyUsage removes a specific usage of a dependency
func (l *Lockfile) RemoveDependencyUsage(moduleName, usedBy string) {
	entry, exists := l.Dependencies[moduleName]
	if !exists {
		return
	}

	// Remove the specific usage
	for i, user := range entry.UsedBy {
		if user == usedBy {
			entry.UsedBy = append(entry.UsedBy[:i], entry.UsedBy[i+1:]...)
			break
		}
	}

	// If no more usages and it's not a direct dependency, remove it entirely
	if len(entry.UsedBy) == 0 && !entry.Direct {
		delete(l.Dependencies, moduleName)
	} else {
		l.Dependencies[moduleName] = entry
	}
}

// GetUnusedDependencies returns a list of dependencies that are no longer used
func (l *Lockfile) GetUnusedDependencies() []string {
	var unused []string

	for moduleName, entry := range l.Dependencies {
		// Skip direct dependencies
		if entry.Direct {
			continue
		}

		// Check if it's still in direct deps
		isDirect := false
		for _, dep := range l.DirectDeps {
			if dep == moduleName {
				isDirect = true
				break
			}
		}

		// If not direct and no usages, it's unused
		if !isDirect && len(entry.UsedBy) == 0 {
			unused = append(unused, moduleName)
		}
	}

	return unused
}

// GetDependencyInfo returns information about a specific dependency
func (l *Lockfile) GetDependencyInfo(moduleName string) (LockfileEntry, bool) {
	entry, exists := l.Dependencies[moduleName]
	return entry, exists
}

// IsDirectDependency checks if a module is a direct dependency
func (l *Lockfile) IsDirectDependency(moduleName string) bool {
	for _, dep := range l.DirectDeps {
		if dep == moduleName {
			return true
		}
	}
	return false
}

// GetDependencyVersion returns the version of a dependency
func (l *Lockfile) GetDependencyVersion(moduleName string) (string, bool) {
	entry, exists := l.Dependencies[moduleName]
	if !exists {
		return "", false
	}
	return entry.Version, true
}

// GetAllDependencies returns all dependencies as a map
func (l *Lockfile) GetAllDependencies() map[string]LockfileEntry {
	result := make(map[string]LockfileEntry)
	for k, v := range l.Dependencies {
		result[k] = v
	}
	return result
}

// GetDirectDependencies returns the list of direct dependencies
func (l *Lockfile) GetDirectDependencies() []string {
	result := make([]string, len(l.DirectDeps))
	copy(result, l.DirectDeps)
	return result
}

// ValidateLockfile validates the lockfile structure
func (l *Lockfile) ValidateLockfile() error {
	// Check that all direct deps exist in dependencies
	for _, directDep := range l.DirectDeps {
		if _, exists := l.Dependencies[directDep]; !exists {
			return fmt.Errorf("direct dependency '%s' not found in dependencies", directDep)
		}
	}

	// Check that all dependencies marked as direct are in direct deps
	for moduleName, entry := range l.Dependencies {
		if entry.Direct {
			found := false
			for _, directDep := range l.DirectDeps {
				if directDep == moduleName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("dependency '%s' marked as direct but not in direct deps", moduleName)
			}
		}
	}

	return nil
}
