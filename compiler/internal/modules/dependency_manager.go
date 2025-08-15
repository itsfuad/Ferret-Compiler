package modules

import (
	"fmt"
)

// DependencyManager handles dependency management using the lockfile system
type DependencyManager struct {
	projectRoot string
	lockfile    *Lockfile
}

// NewDependencyManager creates a new dependency manager for the given project
func NewDependencyManager(projectRoot string) (*DependencyManager, error) {
	lockfile, err := LoadLockfile(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to load lockfile: %w", err)
	}

	return &DependencyManager{
		projectRoot: projectRoot,
		lockfile:    lockfile,
	}, nil
}
