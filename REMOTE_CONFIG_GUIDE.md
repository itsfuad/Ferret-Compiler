# Remote Module Configuration in Ferret

## Overview

Ferret now supports **explicit control** over remote module imports and sharing through the `[remote]` section in `fer.ret`. This provides security and control over what modules can be imported and whether your modules can be shared with others.

## Configuration Options

### `enabled` - Controls Remote Import Permissions

Controls whether **this project** can import remote modules from external sources.

```toml
[remote]
enabled = true   # Allow importing remote modules
# enabled = false # Disable remote imports (default for security)
```

### `share` - Controls Module Sharing Permissions

Controls whether **other projects** can import this project as a remote module.

```toml
[remote] 
enabled = true
share = true     # Allow others to import this project
# share = false  # Prevent others from importing this project (default)
```

## How It Works

### 1. Remote Import Control (`enabled`)

When you try to import a remote module:

```rust
import "github.com/user/awesome-lib/utils";
```

**If `enabled = false`:**
```
Error: remote module imports are disabled in this project

To enable remote imports, set 'enabled = true' in the [remote] section of fer.ret:

[remote]
enabled = true
share = false
```

**If `enabled = true`:**
- Import proceeds normally through the module resolution system

### 2. Remote Sharing Control (`share`)

When someone tries to import **your project** as a remote module:

**If `share = false` (default):**
```
Error: module 'github.com/youruser/yourproject' does not allow remote sharing

The module owner has set 'share = false' in their fer.ret file.
Contact the module maintainer to enable sharing or use a different module.
```

**If `share = true`:**
- Other projects can successfully import and use your modules

## Module Validation Requirements

### Valid Ferret Module Structure

For a module to be importable, it **must** have a proper `fer.ret` file:

```
your-module/
├── fer.ret          # Required: Module configuration
├── src/
│   └── main.fer     # Your module code
└── README.md        # Optional: Documentation
```

### Required fer.ret Structure

```toml
[default]
name = "your-module"
version = "1.0.0"

[compiler]
version = "0.1.0"

[remote]
enabled = true    # If this module imports other modules
share = true      # Required: Allow others to import this module

[dependencies]
# Any dependencies this module needs
github.com/other/lib = "^v1.0.0"
```

### Automatic Dependency Resolution

When you install a remote module:

1. **Download module** → Validate fer.ret exists
2. **Check sharing permission** → Ensure `share = true`
3. **Read dependencies** → Parse [dependencies] section
4. **Install recursively** → Download all required dependencies
5. **Flat structure** → Store all modules at same level

**Example Flow:**
```bash
ferret get github.com/user/web-framework

# Downloads and validates:
# ├── github.com/user/web-framework@v2.1.0/     # Main module
# ├── github.com/user/http-utils@v1.5.0/       # Dependency 1
# ├── github.com/other/json-parser@v3.0.1/     # Dependency 2
# └── github.com/other/logger@v1.2.0/          # Dependency 3
```

## Error Scenarios

### Missing fer.ret File
```
Error: module 'github.com/user/legacy-lib' is not a valid Ferret module

The module does not contain a 'fer.ret' file, which is required for:
- Dependency management
- Module configuration  
- Version compatibility

Please use modules that follow Ferret's module structure or contact the maintainer to add fer.ret support.
```

### Invalid fer.ret File
```
Error: module 'github.com/user/broken-lib' has an invalid fer.ret file

Error reading configuration: invalid TOML syntax at line 5

Please contact the module maintainer to fix the fer.ret file.
```

### Sharing Disabled
```
Error: module 'github.com/user/private-lib' does not allow remote sharing

The module owner has set 'share = false' in their fer.ret file.
Contact the module maintainer to enable sharing or use a different module.
```

## Usage Examples

### 1. Private Project (Default)
```toml
[remote]
enabled = false  # Don't import any remote modules
share = false    # Don't allow others to import this project
```

### 2. Consumer Project
```toml
[remote]
enabled = true   # Can import remote modules
share = false    # But others can't import this project
```

### 3. Library Project
```toml
[remote] 
enabled = true   # Can import dependencies
share = true     # Allow others to use this as a library
```

### 4. Secure Enterprise Project
```toml
[remote]
enabled = false  # No external dependencies allowed
share = false    # No external access to this code
```

## Security Benefits

1. **Explicit Consent**: Remote imports must be explicitly enabled
2. **Supply Chain Control**: Prevents accidental external dependencies
3. **Code Protection**: Prevents unauthorized use of proprietary code
4. **Audit Trail**: Clear configuration shows import/export policies

## Commands

### Initialize Project with Remote Settings
```bash
ferret init
# You'll be prompted:
# "Do you want to allow remote module import? [Yes|No|Y|N] (default: no)"
# "Do you want to allow sharing your modules to others? [Yes|No|Y|N] (default: no)"
```

### Enable Remote Imports for Existing Project
```bash
# Edit fer.ret manually or reinitialize
[remote]
enabled = true
share = false
```

### Test Remote Import Settings
```bash
# Try importing a remote module
ferret your-file.fer

# If disabled, you'll see a clear error with instructions
# If enabled, imports work normally
```

## Best Practices

1. **Start Secure**: Default to `enabled = false` for new projects
2. **Enable Explicitly**: Only enable remote imports when needed
3. **Share Intentionally**: Only set `share = true` for public libraries
4. **Document Dependencies**: Keep fer.ret in version control
5. **Review Regularly**: Audit remote settings periodically

## Backwards Compatibility

- **Missing [remote] section**: Treated as `enabled = false, share = false` for security
- **Existing projects**: Continue working, but remote imports require explicit enabling
- **⚠️ BREAKING**: Modules without fer.ret are now **rejected** for dependency management
- **Migration**: Legacy modules need to add fer.ret with proper dependency declarations

## Implementation Notes

- **Import Check**: Performed before dependency resolution
- **Share Check**: Performed after downloading but before caching
- **Dependency Resolution**: Automatically installs module dependencies from fer.ret
- **Error Recovery**: Failed validations remove downloaded files
- **Fallback**: Missing configurations default to secure settings
- **⚠️ Validation**: Modules **must** have valid fer.ret files to be used 