# Ferret Lockfile System Guide

## Overview

The Ferret compiler now uses a sophisticated lockfile system for dependency management. This system provides:

- **Flat dependency structure**: All dependencies (direct and indirect) are stored in a single JSON file
- **Usage tracking**: The system tracks which modules depend on which other modules
- **Automatic cleanup**: Unused dependencies can be automatically removed
- **Version pinning**: Exact versions are locked to ensure reproducible builds

## Lockfile Structure

The lockfile (`ferret.lock`) is a JSON file with the following structure:

```json
{
  "version": "1.0",
  "direct_deps": [
    "github.com/user/repo",
    "github.com/another/module"
  ],
  "dependencies": {
    "github.com/user/repo": {
      "version": "v1.0.0",
      "direct": true,
      "used_by": [],
      "description": "Main dependency"
    },
    "github.com/user/transitive": {
      "version": "v2.0.0",
      "direct": false,
      "used_by": ["github.com/user/repo"],
      "description": ""
    }
  },
  "generated_at": "2024-01-15T10:30:00Z"
}
```

## Key Concepts

### Direct vs Indirect Dependencies

- **Direct dependencies**: Modules explicitly added by the user via `ferret get`
- **Indirect dependencies**: Modules required by direct dependencies (transitive dependencies)

### Usage Tracking

The system tracks which modules depend on which other modules through the `used_by` field. This allows for:

- Identifying unused dependencies
- Understanding dependency relationships
- Safe removal of dependencies

## Commands

### `ferret get [module]`

Installs a direct dependency and its transitive dependencies.

```bash
# Install a specific module
ferret get github.com/user/repo@v1.0.0

# Install all dependencies from fer.ret
ferret get
```

### `ferret remove [module]`

Removes a direct dependency and cleans up unused indirect dependencies.

```bash
ferret remove github.com/user/repo
```

### `ferret list`

Lists all dependencies with their status (direct vs indirect).

```bash
ferret list
```

### `ferret cleanup`

Removes unused dependencies that are no longer needed.

```bash
ferret cleanup
```

## Example Workflow

### 1. Initialize a Project

```bash
ferret init myproject
cd myproject
```

### 2. Add Dependencies

```bash
# Add a direct dependency
ferret get github.com/user/repo@v1.0.0

# This will create a lockfile like:
{
  "version": "1.0",
  "direct_deps": ["github.com/user/repo"],
  "dependencies": {
    "github.com/user/repo": {
      "version": "v1.0.0",
      "direct": true,
      "used_by": [],
      "description": ""
    },
    "github.com/user/transitive": {
      "version": "v2.0.0",
      "direct": false,
      "used_by": ["github.com/user/repo"],
      "description": ""
    }
  }
}
```

### 3. Add Another Dependency

```bash
ferret get github.com/another/module@v1.5.0
```

### 4. List Dependencies

```bash
ferret list
# Output:
# Dependencies:
# ============
# Direct dependencies:
#   github.com/user/repo@v1.0.0
#   github.com/another/module@v1.5.0
# 
# Indirect dependencies:
#   github.com/user/transitive@v2.0.0 (used by: github.com/user/repo)
```

### 5. Remove a Dependency

```bash
ferret remove github.com/user/repo
# This will also remove github.com/user/transitive if it's no longer used
```

## Benefits

### 1. Reproducible Builds

The lockfile ensures that everyone working on the project gets exactly the same versions of dependencies.

### 2. Dependency Tracking

The system automatically tracks which modules depend on which other modules, making it easy to understand the dependency graph.

### 3. Automatic Cleanup

When you remove a dependency, the system automatically identifies and removes unused transitive dependencies.

### 4. Flat Structure

All dependencies are stored in a single, easy-to-read JSON file, making it simple to understand what's installed.

## Migration from Old System

If you have an existing project using the old dependency system:

1. The system will automatically create a lockfile when you run `ferret get`
2. Existing dependencies in `fer.ret` will be migrated to the lockfile
3. The `fer.ret` file will continue to contain only direct dependencies
4. The lockfile will contain the complete dependency graph

## Security Features

The lockfile system maintains all existing security features:

- Remote module sharing controls
- Version validation
- Dependency verification

## File Locations

- **Lockfile**: `ferret.lock` (in project root)
- **Cache**: `.ferret/modules/` (in project root)
- **Configuration**: `fer.ret` (in project root)

## Best Practices

1. **Commit the lockfile**: Always commit `ferret.lock` to version control
2. **Don't edit manually**: Let the system manage the lockfile
3. **Use cleanup**: Run `ferret cleanup` periodically to remove unused dependencies
4. **Check dependencies**: Use `ferret list` to understand your dependency graph

## Troubleshooting

### Lockfile Corruption

If the lockfile becomes corrupted:

```bash
# Remove the lockfile
rm ferret.lock

# Reinstall all dependencies
ferret get
```

### Missing Dependencies

If dependencies are missing from cache:

```bash
# Reinstall all dependencies
ferret get
```

### Version Conflicts

If you encounter version conflicts:

1. Check the lockfile for conflicting versions
2. Use `ferret remove` to remove conflicting dependencies
3. Reinstall with specific versions using `ferret get` 