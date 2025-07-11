# Ferret Programming Language

[![CI](https://github.com/itsfuad/Ferret-Programming-Language/actions/workflows/ci.yml/badge.svg)](https://github.com/itsfuad/Ferret-Programming-Language/actions/workflows/ci.yml)
[![Release](https://github.com/itsfuad/Ferret-Programming-Language/actions/workflows/release.yml/badge.svg)](https://github.com/itsfuad/Ferret-Programming-Language/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsfuad/Ferret-Programming-Language)](https://goreportcard.com/report/github.com/itsfuad/Ferret-Programming-Language)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

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
   cd compiler
   go build -o ferret cmd/main.go
   ```

### Usage

#### Initialize a new Ferret project
```bash
# Initialize in current directory
ferret init

# Initialize in specific directory
ferret init /path/to/project
```

This creates a `.ferret.json` configuration file with default settings.

#### Compile and run Ferret code
```bash
# Compile a Ferret file
ferret filename.fer

# Compile with debug output
ferret filename.fer --debug

# Debug flag can be placed anywhere
ferret --debug filename.fer
```

#### Help
```bash
ferret
# Output: Usage: ferret <filename> [--debug] | ferret init [path]
```

### Project Configuration
The `.ferret.json` file contains project-specific settings:

```json
{
  "compiler": {
    "version": "0.1.0"
  },
  "cache": {
    "path": ".ferret/modules"
  },
  "remote": {
    "enabled": true,
    "share": false
  },
  "dependencies": {}
}
```

## Key Features
- Statically Typed: Strong typing ensures that errors are caught early, making your code more predictable and robust.
- Beginner-Friendly: Ferret's syntax is designed to be easy to read and understand, even for new developers.
- Expressive Code: With simple syntax and clear semantics, Ferret is made to be highly expressive without unnecessary complexity.
- First-Class Functions: Functions are treated as first-class citizens, enabling functional programming paradigms while maintaining simplicity.
- Clear Structs and Interfaces: Structs have methods and are used for simplicity, with implicit interface implementation for cleaner code.

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

### Arrays
```rs
// Array declarations
let arr1: []i32;                         // Integer array
let arr2: []str = ["hello", "world"];    // String array with initialization
let arr2d: [][]i32 = [[1, 2], [3, 4]];  // 2D array

// Array operations
arr1[0] = 10;      // Assignment
a = arr1[0];       // Access
```

### Structs
```rs
// Named struct type declaration
type Point struct {
    x: i32,
    y: i32
};

// Creating struct instances
let point: Point = @Point{x: 10, y: 20};

// Anonymous struct type
let user = struct {
    name: str,
    age: i32
};

// Anonymous struct with initialization
let person: struct {
    name: str,
    age: i32
} = @struct{name: "John", age: 20};

// Nested structs
type User struct {
    name: str,
    address: struct {
        street: str,
        city: str
    }
};
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

#### Project Structure
```
Ferret_Compiler/
├── compiler/           # Go-based compiler implementation
│   ├── cmd/           # CLI entry point and argument parsing
│   ├── colors/        # Terminal color output utilities
│   ├── ctx/           # Compiler context management
│   └── internal/
│       ├── config/    # Project configuration (.ferret.json)
│       ├── frontend/  # Lexer, parser, and AST
│       ├── semantic/  # Symbol resolution and type checking
│       ├── source/    # Source code location tracking
│       ├── report/    # Error reporting system
│       ├── types/     # Type system definitions
│       └── utils/     # Utility functions
├── app/               # Sample Ferret programs
│   ├── cmd/          # Main application files
│   ├── data/         # Data modules
│   └── maths/        # Math utility modules
├── scripts/           # Build and development scripts
│   ├── build.*       # Compiler build scripts
│   ├── test.*        # Test execution scripts
│   ├── fmt.*         # Code formatting scripts
│   ├── run.*         # Quick development test scripts
│   ├── ci-check.*    # Local CI validation scripts
│   ├── pack.*        # Extension packaging scripts
│   └── README.md     # Script documentation
└── .github/workflows/ # CI/CD pipeline definitions
```

## Type Declarations
```rs
// Type aliases
type Integer i32;
type Text str;

// Struct types
type Point struct {
    x: i32,
    y: i32
};

// Array types
type IntArray []i32;
type Matrix [][]f32;
```

## Roadmap
- [x] Basic syntax
- [x] Tokenizer
- [x] Parser
- [x] Variable declaration and assignment
- [x] Expressions
- [x] Unary operators
- [x] Increment/Decrement operators
- [x] Assignment operators
- [x] Grouping
- [x] Arrays
    - [x] Array indexing
    - [x] Array assignment
- [x] Structs
    - [x] Anonymous structs
    - [x] Struct literals
    - [x] Struct field access
    - [x] Struct field assignment
- [x] Methods
- [ ] Interfaces
- [x] Functions
- [x] Conditionals
- [ ] Loops (for, while)
- [ ] Switch statements
- [ ] Type casting
- [ ] Maps
- [ ] Range expressions
- [ ] Error handling
- [ ] Imports and modules
- [ ] Nullable/optional types
- [ ] Generics
- [ ] Advanced code generation
- [x] Rich error reporting
- [ ] Branch analysis
- [x] Resolver
- [x] Type checking

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

## License
This project is licensed under the Mozilla Public License 2.0 - see the LICENSE file for details.
