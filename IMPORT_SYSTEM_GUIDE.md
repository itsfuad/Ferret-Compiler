# Ferret Import System - New Workflow

## Overview

The Ferret import system now works like Go's module system with these key principles:

1. **Use `ferret get` to install dependencies** (like `go get`)
2. **Auto-populate fer.ret** when installing (like go.mod)
3. **Auto-download missing cache** when fer.ret/ferret.lock exist
4. **Flat dependency structure** for better version management

## Workflow Examples

### 1. Installing a New Module

```bash
# Install a new remote module
ferret get github.com/user/awesome-lib

# This automatically:
# - Downloads the module 
# - Adds it to fer.ret [dependencies] section
# - Updates ferret.lock with version info
# - Stores in flat cache: .ferret/modules/github.com/user/awesome-lib@v1.2.3/
```

**Auto-generated fer.ret:**
```toml
[dependencies]
github.com/user/awesome-lib = "^v1.2.3"
```

**Auto-generated ferret.lock:**
```json
{
  "version": "1.0",
  "generated_at": "2024-01-15T10:30:00Z",
  "packages": {
    "github.com/user/awesome-lib@v1.2.3": {
      "version": "v1.2.3",
      "resolved_url": "https://github.com/user/awesome-lib/archive/refs/tags/v1.2.3.zip",
      "checksum": "sha256:abc123...",
      "downloaded_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### 2. Using Installed Modules

```rust
// In your .fer file
import "github.com/user/awesome-lib/utils";

// Use the imported functionality
let result = utils::process_data("hello");
```

### 3. Missing Cache Auto-Recovery

**Scenario**: You have fer.ret and ferret.lock but someone deleted .ferret/cache/

```bash
# Just run your code normally
ferret main.fer

# Output:
# Module github.com/user/awesome-lib@v1.2.3 was previously installed but cache is missing. Re-downloading...
# Successfully auto-installed github.com/user/awesome-lib@v1.2.3
# [compilation continues normally]
```

### 4. Missing Module Error (Improved)

**Before:**
```
Error: remote module not found in cache: github.com/user/new-lib
Run: ferret get github.com/user/new-lib
```

**After:**
```
Error: module 'github.com/user/new-lib' is not installed

To install this module, run:
  ferret get github.com/user/new-lib

This will automatically add it to fer.ret and download it to cache.
```

## Cache Structure (Flat)

### Old Structure (Nested):
```
.ferret/modules/
└── github.com/
    └── user/
        └── repo@v1.0.0/
            └── dependencies/
                └── dep@v2.0.0/
```

### New Structure (Flat):
```
.ferret/modules/
├── github.com/user/repo@v1.0.0/
├── github.com/user/repo@v1.1.0/          # Multiple versions supported
├── github.com/other/awesome-lib@v2.3.1/
└── github.com/other/dep@v1.5.0/
```

## Commands

### Install Dependencies
```bash
# Install specific module
ferret get github.com/user/module

# Install all dependencies from fer.ret
ferret get
```

### Remove Dependencies
```bash
# Remove module (updates fer.ret, ferret.lock, and cache)
ferret remove github.com/user/module
```

### Initialize Project
```bash
# Create new project with fer.ret
ferret init
```

## Benefits

1. **Go-like Experience**: Familiar workflow for developers
2. **Auto-Recovery**: Missing cache is automatically restored
3. **Version Isolation**: Multiple versions of same module can coexist
4. **Clear Error Messages**: Helpful guidance for users
5. **Automatic Management**: fer.ret and ferret.lock are maintained automatically

## Migration from Old System

The new system is backwards compatible. Existing projects will continue to work, but new installs will use the flat structure and improved workflow. 