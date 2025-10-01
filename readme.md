# Ferret Programming Language (In Development)

![Discord Shield](https://discord.com/api/guilds/1243622698551345153/widget.png?style=shield)
[![CI](https://github.com/itsfuad/Ferret-Compiler/actions/workflows/ci.yml/badge.svg)](https://github.com/itsfuad/Ferret-Compiler/actions/workflows/ci.yml)
[![Release](https://github.com/itsfuad/Ferret-Compiler/actions/workflows/release.yml/badge.svg)](https://github.com/itsfuad/Ferret-Compiler/actions/workflows/release.yml)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

![Ferret Mascot](ferret.png)

Welcome to Ferret! Ferret is a statically typed, beginner-friendly programming language designed to bring clarity, simplicity, and expressiveness to developers. With a focus on readability and a clean syntax, Ferret makes it easier to write clear, maintainable code while embracing modern programming principles.

## Quick Start

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/itsfuad/Ferret-Compiler.git
   cd Ferret-Compiler
   ```

2. Build the compiler:
   ```bash
   cd scripts
   ./build.sh    # Linux/macOS/Git Bash
   ./build.bat   # Windows CMD/PowerShell
   ```

3. Add the `bin` directory to your PATH environment variable to use `ferret` commands globally.

## Language Support Extension
Download the Ferret Language Support Extension for your IDE for Syntax Highlighting and Language Server Protocol (LSP) support.

### VS Code Extension Features
The Ferret VS Code extension provides comprehensive language support including:

- **Syntax Highlighting**: Full syntax highlighting for `.fer`, `.ret`, and `.lock` files
- **Language Server Protocol (LSP)**: Advanced IDE features powered by the Ferret LSP server
- **Real-time Diagnostics**: Error and warning reporting as you type
- **Code Completion**: Smart autocomplete with snippets for Ferret language constructs
- **Hover Information**: Documentation and type information on hover
- **Go-to-Definition**: Navigate to symbol definitions (framework ready)
- **Find References**: Find all references to symbols (framework ready)
- **Document Symbols**: Outline view and symbol navigation (framework ready)
- **Code Formatting**: Automatic code formatting (framework ready)

### LSP Server Installation & Usage

#### Building the LSP Server
```bash
cd scripts
./lsp.sh      # Linux/macOS/Git Bash  
./lsp.bat     # Windows CMD/PowerShell
```

This creates the `ferret-lsp` binary in the `bin` directory.

#### VS Code Extension Installation
1. Build the extension:
   ```bash
   cd scripts
   ./pack.sh     # Linux/macOS/Git Bash
   ./pack.bat    # Windows CMD/PowerShell
   ```

2. Install the generated `.vsix` file in VS Code:
   - Open VS Code
   - Go to Extensions (Ctrl+Shift+X)
   - Click "..." menu → "Install from VSIX..."
   - Select the generated `ferret-*.vsix` file

#### LSP Configuration
Configure the Ferret Language Server in VS Code settings:

```json
{
  "ferretLanguageServer.enabled": true,
  "ferretLanguageServer.serverPath": "",
  "ferretLanguageServer.debug": false,
  "ferretLanguageServer.completion.enabled": true,
  "ferretLanguageServer.hover.enabled": true,
  "ferretLanguageServer.definition.enabled": true,
  "ferretLanguageServer.diagnostics.enabled": true,
  "ferretLanguageServer.trace.server": "off"
}
```

#### Troubleshooting

**LSP Server Not Starting:**

1. **Automatic Detection**: The extension will try to find `ferret-lsp` in these locations:
   - `workspace/bin/ferret-lsp`
   - `workspace/../bin/ferret-lsp`
   - `ferret-lsp` in system PATH

2. **Manual Configuration**: If auto-detection fails, set the path manually:
   - Open VS Code settings (Ctrl+,)
   - Search for "ferretLanguageServer.serverPath"
   - Set the full path to your `ferret-lsp` executable
   - Or use the command: **Ferret: Set LSP Server Path**

3. **Build the LSP Server**: If the binary doesn't exist:
   ```bash
   cd Ferret-Compiler/scripts
   ./lsp.sh      # Linux/macOS
   ./lsp.bat     # Windows
   ```

4. **Check LSP Status**: Use these commands from the VS Code Command Palette:
   - **Ferret: Show LSP Output** - View detailed logs
   - **Ferret: Restart LSP Server** - Restart if needed
   - **Ferret: Toggle LSP Server** - Enable/disable

#### Available Commands
- **Ferret: Toggle LSP Server** - Enable/disable the language server
- **Ferret: Restart LSP Server** - Restart the language server
- **Ferret: Show LSP Output** - View language server logs

### Usage

#### Initialize a new Ferret project
```bash
# Initialize in current directory
ferret init

# Initialize in specific directory
ferret init /path/to/project
```

This creates a `fer.ret` configuration file with default settings.
#### Help
```bash
ferret

Ferret - A statically typed, beginner-friendly programming language

USAGE:
  ferret run [options]                 Run project using entry point from fer.ret

MODULE MANAGEMENT:
  ferret init [path]                   Initialize a new Ferret project
  ferret get <module>                  Install a module dependency
  ferret update [module]               Update module(s) to latest version
  ferret remove <module>               Remove a module dependency
  ferret list                          List all installed modules
  ferret sniff                         Check for available module updates
  ferret orphan                        List orphaned cached modules
  ferret clean                         Remove unused module cache

OPTIONS:
  -d, -debug                           Enable debug mode

NOTE: All flags support both single dash (-flag) and double dash (--flag) formats

EXAMPLES:
  ferret run                           Run project using fer.ret configuration
  ferret run -debug                    Run with debug output
  ferret run --debug                   Run with debug output (alternative)
  ferret init my-project-name          Create new project named my-project-name
  ferret get github.com/user/module    Install a module from GitHub
  ferret update                        Update all modules
```

### Project Configuration
The `fer.ret` file contains project-specific settings:

```toml
name = "your_project_name"

[compiler]
version = "0.0.1"

[build]
entry = "path/to/your/entry.fer"
output = "bin"

[cache]
path = ".ferret"

[remote]
enabled = true
share = false

[neighbor]
modulename = "../path/to/your/module"

[dependencies]
github.com/user/repo = "0.0.2"
```

## Module Management

Ferret provides a comprehensive module management system for handling dependencies with version control.

### Basic Module Commands

#### Initialize a Project
```bash
# Initialize in current directory
ferret init

# Initialize in specific directory
ferret init /path/to/project
```

#### Install Dependencies
```bash
# Install all dependencies from fer.ret
ferret get

# Install specific module
ferret get github.com/user/repo

# Install specific version
ferret get github.com/user/repo@v1.0.0
```

#### Check for Updates
```bash
# Check which dependencies have updates available
ferret sniff

# Example output:
# 📦 github.com/user/repo: v1.0.0 → v1.2.0 (update available)
# ✅ github.com/other/module: v2.1.0 (up to date)
```

#### Update Dependencies
```bash
# Update all dependencies to latest versions
ferret update

# Update specific module to latest version
ferret update github.com/user/repo
```

#### Manage Dependencies
```bash
# List all dependencies (direct and transitive)
ferret list

# Remove a dependency
ferret remove github.com/user/repo

# See orphan modules
ferret orphan

# Clean up unused dependencies
ferret cleanup
```

### Dependency Resolution Strategy

Ferret follows a **dependency-driven update strategy** that ensures compatibility:

1. **Direct Dependencies**: Only modules explicitly listed in `fer.ret` are updated by user commands
2. **Transitive Dependencies**: Automatically resolved to the exact versions specified by their parent modules
3. **Version Pinning**: Transitive dependencies use their specified versions, not the latest available versions

#### Example Scenario
```bash
# Your project uses ModuleA@v1.0
# ModuleA@v1.0 depends on ModuleB@v2.0
# ModuleB@v3.0 is available but won't be used

ferret sniff  # Shows: "ModuleA: v1.0 → v1.2 (update available)"
ferret update     # Updates ModuleA to v1.2
                  # If ModuleA@v1.2 uses ModuleB@v2.1, then ModuleB gets updated
                  # If ModuleA@v1.2 still uses ModuleB@v2.0, then ModuleB stays at v2.0
```

This prevents **dependency hell** by ensuring that modules are only updated to versions that are tested and compatible with their parent dependencies.

### Module Configuration

#### Remote Module Settings
Enable remote module imports in `fer.ret`:
```toml
[remote]
enabled = true    # Allow downloading modules from GitHub
share = false     # Whether to share your modules publicly
```

#### Security Features
- **Explicit Opt-in**: Remote imports must be explicitly enabled
- **Version Verification**: All module versions are verified against GitHub releases
- **Lockfile System**: Exact dependency versions are tracked in `ferret.lock`
- **Cache Management**: Downloaded modules are cached locally for faster builds

### Advanced Usage

#### Version Specifications
```bash
# Latest version
ferret get github.com/user/repo@latest

# Specific version
ferret get github.com/user/repo@v1.0.0

# Version without 'v' prefix (auto-detected)
ferret get github.com/user/repo@1.0.0
```

#### Project Structure
```
myproject/
├── fer.ret          # Project configuration
├── ferret.lock      # Dependency lockfile (auto-generated)
├── .ferret/         # Cache directory
└── src/             # Your Ferret source files
```

#### Lockfile System
The `ferret.lock` file tracks:
- Exact versions of all dependencies (direct and transitive)
- Dependency relationships and usage tracking
- Cache paths and download metadata
- Generation timestamp and integrity checks

## Key Features
- **Statically Typed**: Strong typing ensures that errors are caught early, making your code more predictable and robust.
- **Beginner-Friendly**: Ferret's syntax is designed to be easy to read and understand, even for new developers.
- **Expressive Code**: With simple syntax and clear semantics, Ferret is made to be highly expressive without unnecessary complexity.
- **First-Class Functions**: Functions are treated as first-class citizens, enabling functional programming paradigms while maintaining simplicity.
- **Clear Structs and Interfaces**: Structs have methods and are used for simplicity, with implicit interface implementation for cleaner code.
- **Advanced Module System**: Comprehensive dependency management with version control, automated resolution, and secure remote module support.
- **Developer-Friendly Tooling**: Rich error reporting, debug mode, and comprehensive CLI commands for project management.

## Basic Syntax

### Variables and Types
```rs
// Single variable with type inference
let x = 10;
let y: f32;
let myname: str = "John";

// Multiple variables with type
let p, q, r: i32, f32, str = 10, 20.0, "hello";
let p, q: i32, str = 10, "hello";

// Multiple variables with type inference
let p, q, r = 10, 20.0, "hello";
let p, q = 10, "hello";

// Assignments
x = 15;                          // Single variable
p, q = 10, "hello";             // Multiple variables
p, q, r = 10, 20.0, "hello";    // Multiple variables with different types
```

### Type Declarations
```rs
// Type aliases
type Integer i32;
type Text str;
```

### Operators
```rs
// Arithmetic operators
a = (a + b) * c;   // Basic arithmetic
x++;               // Postfix increment
x--;               // Postfix decrement
++x;               // Prefix increment
--x;               // Prefix decrement

// Assignment operators
a += b;            // Add and assign
a -= b;            // Subtract and assign
a *= b;            // Multiply and assign
a /= b;            // Divide and assign
a %= b;            // Modulo and assign
```

### Type casting
```rs
// Type casting
let a: i32 = 10;
let b: f32 = a as f32; // Cast i32 to f32
```

## Compiler Architecture

The Ferret compiler follows a multi-stage compilation pipeline designed for maintainability and extensibility:

### Frontend
- **Lexer**: Converts source code into tokens for parsing
- **Parser**: Transforms tokens into an Abstract Syntax Tree (AST)
- **AST**: Represents the program structure in a tree format

### Semantic Analysis
- **Collector**: Gathers symbols and builds symbol tables for scope resolution
- **Resolver**: Resolves symbol references and validates module imports
- **Type Checker**: Performs type inference and validates type compatibility

### Backend
- **Code Generator**: Translates the validated AST into target assembly code (currently x86-64)

## Roadmap
### Language System
- [x] Basic syntax
- [x] Tokenizer
- [x] Parser
- [x] Variable declaration and assignment
- [x] Simple assignments
- [x] Multiple assignments
- [x] Expressions
- [x] Unary operators
- [x] Increment/Decrement operators
- [x] Assignment operators
- [x] Grouping
- [x] Type aliases
- [x] Arrays
    - [x] Array indexing
    - [x] Array assignment
    - [x] String indexing
    - [x] Array spreading
- [x] Structs
    - [x] Anonymous structs
    - [x] Struct literals
    - [x] Struct field access
    - [x] Struct field assignment
    - [x] Methods
- [x] Interfaces
- [x] Functions
    - [x] Dead code elimination
    - [x] Function calls
    - [x] Function parameters
    - [x] Variadic parameters
    - [x] Function return values
    - [x] Annonymous functions
    - [x] Closures
- [x] Conditionals
    - [x] Control flow analysis
- [ ] Loops (for, while)
- [ ] Switch statements
- [x] Type casting
- [ ] Maps
- [ ] Range expressions
- [ ] Error handling
- [x] Imports and modules
    - [x] Local imports
    - [x] Remote module imports (GitHub)
    - [x] Module dependency resolution
    - [x] Version management and lockfiles
    - [x] Dependency update detection
    - [x] Transitive dependency handling
    - [x] Module caching system
    - [x] Security controls for remote imports
- [ ] Nullable/optional types
- [ ] Generics
- [ ] Advanced code generation
- [x] Rich error reporting
- [x] Collector (Symbol collection and scope building)
- [x] Resolver (Symbol resolution and reference binding)
- [x] Type checking (Type inference and validation)
- [ ] Code generation (x86-64 assembly - basic implementation)

### Dependency Management
- [x] Read dependency from config file
- [x] Download dependencies from config file
  - [x] Download indirect dependencies
- [x] Cache downloaded modules
- [x] Update config file
- [x] Update lockfile
- [x] Update dependency
- [x] Delete dependencies
- [x] Check for updates
- [x] Auto update
- [x] Detect orphans
- [x] Delete orphans
- [ ] Compiler version check for all modules

## Contributing
Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development
To work on the Ferret compiler:

1. **Prerequisites**: Go 1.19 or later
2. **Clone the repository** and navigate to the compiler directory
3. **Run tests**:
   ```bash
   # Run all tests
   go test ./...
   
   # Run tests with verbose output
   go test -v ./...
   
   # Run specific test package
   go test ./cmd -v
   ```

4. **Build and test locally**:
   ```bash
   # Build the compiler
   go build -o ferret cmd/main.go
   
   # Test with sample files
   ./ferret ../app/cmd/main.fer --debug
   ```

5. **Run local CI checks** (recommended before pushing):
   ```bash
   # On Linux/macOS/Git Bash
   ./scripts/ci-check.sh
   
   # On Windows Command Prompt/PowerShell
   .\scripts\ci-check.bat
   ```
   This script runs the same checks as the CI pipeline locally.

### Development Scripts
The project includes several convenience scripts in the `scripts/` directory:

```bash
# Build the compiler
./scripts/build.sh        # Linux/macOS/Git Bash
.\scripts\build.bat        # Windows CMD/PowerShell

# Run tests with formatted output
./scripts/test.sh          # Linux/macOS/Git Bash
.\scripts\test.bat         # Windows CMD/PowerShell

# Format all code
./scripts/fmt.sh           # Linux/macOS/Git Bash
.\scripts\fmt.bat          # Windows CMD/PowerShell

# Quick test with sample file
./scripts/run.sh           # Linux/macOS/Git Bash
.\scripts\run.bat          # Windows CMD/PowerShell

# Full CI validation
./scripts/ci-check.sh      # Linux/macOS/Git Bash
.\scripts\ci-check.bat     # Windows CMD/PowerShell
```

See `scripts/README.md` for detailed script documentation.

### Testing
The project includes comprehensive tests for:
- CLI argument parsing
- Lexical analysis (tokenizer)
- Syntax parsing
- Type checking
- Semantic analysis
- Configuration management

Run the test suite before submitting contributions:
```bash
cd compiler
go test ./...
```

### CI/CD Pipeline
The project uses GitHub Actions for continuous integration and deployment:

#### Automated Workflows
- **CI Pipeline** (`ci.yml`): Runs on all branches and pull requests
  - Executes all tests
  - Checks code formatting with `gofmt`
  - Runs `go vet` for static analysis
  - Builds the compiler

- **Pull Request Validation** (`pr.yml`): Additional checks for PRs to main
  - Comprehensive test suite
  - Code formatting validation
  - Security scanning with gosec
  - CLI functionality testing

- **Release Pipeline** (`release.yml`): Triggers on pushes to main branch
  - Runs full test suite and formatting checks
  - Builds cross-platform binaries (Linux, Windows, macOS)
  - Creates GitHub releases with auto-generated changelog
  - Uploads compiled binaries as release assets

- **Auto-formatting** (`format.yml`): Manual/scheduled code formatting
  - Can be triggered manually via GitHub Actions
  - Automatically formats code using `gofmt`
  - Commits formatting changes if needed

#### Release Process
1. Push changes to main branch
2. All tests must pass
3. Code must be properly formatted
4. Automated release created with version tag
5. Binaries built for multiple platforms
6. Release notes auto-generated from commits

## Development & Contributing

### LSP System Development

The Ferret compiler includes a comprehensive Language Server Protocol (LSP) implementation that provides IDE integration for development tools.

#### Architecture Overview
```
Ferret LSP System Architecture:

VS Code Extension (TypeScript)
        ↓ (TCP Connection)
Ferret LSP Server (Go)
        ↓ (API Calls)
Ferret Compiler Core (Go)
        ↓ (Analysis)
AST → Semantic Analysis → Diagnostics
```

#### LSP Server Components
- **`lsp/main.go`**: Main LSP server implementation with TCP communication
- **`lsp/wio/`**: URI and file path conversion utilities
- **`compiler/cmd/lsp_api.go`**: Compiler integration API for LSP

#### VS Code Extension Components
- **`extension/src/client.ts`**: Main extension client with LSP communication
- **`extension/package.json`**: Extension manifest with capabilities and configuration
- **`extension/syntaxes/`**: Syntax highlighting definitions

#### Development Workflow

1. **Setup Development Environment**:
   ```bash
   # Clone the repository
   git clone https://github.com/itsfuad/Ferret-Compiler.git
   cd Ferret-Compiler
   
   # Build all components
   cd scripts
   ./pack.sh  # Builds compiler, LSP server, and VS Code extension
   ```

2. **LSP Server Development**:
   ```bash
   # Build only the LSP server
   ./scripts/lsp.sh
   
   # Run LSP tests
   cd lsp && go test -v
   
   # Test manually with port binding
   ./bin/ferret-lsp --port 9999
   ```

3. **VS Code Extension Development**:
   ```bash
   cd extension
   
   # Install dependencies
   npm install
   
   # Compile TypeScript
   npm run compile
   
   # Bundle for production
   npm run bundle
   
   # Package extension
   npx vsce package
   ```

4. **Testing LSP Features**:
   ```bash
   # Run all tests
   ./scripts/test.sh
   
   # Test with sample Ferret files
   cd app
   # Create .fer files and test with VS Code extension
   ```

#### Extending LSP Capabilities

To add new LSP features:

1. **Add LSP Method Handler** (in `lsp/main.go`):
   ```go
   func handleYourFeature(writer *bufio.Writer, req Request) {
       // Parse request parameters
       // Call compiler API if needed
       // Send response
   }
   ```

2. **Register in Switch Statement**:
   ```go
   case "textDocument/yourFeature":
       handleYourFeature(writer, req)
   ```

3. **Update Capabilities** (in `handleInitialize`):
   ```go
   "yourFeatureProvider": true,
   ```

4. **Add Tests**:
   ```go
   func TestHandleYourFeature(t *testing.T) {
       // Test implementation
   }
   ```

#### Build Scripts
- **`scripts/build.sh`**: Build main compiler
- **`scripts/lsp.sh`**: Build LSP server only
- **`scripts/pack.sh`**: Build everything (compiler + LSP + extension)
- **`scripts/test.sh`**: Run all tests
- **`scripts/ci-check.sh`**: Full CI validation

#### Configuration & Settings
The LSP system supports extensive configuration:

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `ferretLanguageServer.enabled` | boolean | true | Enable/disable LSP |
| `ferretLanguageServer.debug` | boolean | false | Debug mode |
| `ferretLanguageServer.port` | number | 0 | LSP server port (0 = dynamic) |
| `ferretLanguageServer.completion.enabled` | boolean | true | Code completion |
| `ferretLanguageServer.hover.enabled` | boolean | true | Hover information |
| `ferretLanguageServer.definition.enabled` | boolean | true | Go-to-definition |
| `ferretLanguageServer.diagnostics.enabled` | boolean | true | Error/warning reporting |
| `ferretLanguageServer.trace.server` | string | "off" | LSP communication tracing |

#### Testing & Quality Assurance
- **Unit Tests**: Go test suite for LSP server (`lsp/main_test.go`)
- **Integration Tests**: End-to-end LSP communication testing
- **Manual Testing**: Use `app/test_lsp.fer` for manual verification
- **CI/CD**: Automated testing in GitHub Actions

## License
This project is licensed under the Mozilla Public License 2.0 - see the LICENSE file for details.
