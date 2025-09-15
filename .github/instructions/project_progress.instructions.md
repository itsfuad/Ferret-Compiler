---
applyTo: '*.go'
---

# Ferret Compiler Project Instructions

## Project Overview
Ferret is a statically typed, beginner-friendly programming language designed for clarity, simplicity, and expressiveness. This compiler is written in Go and focuses on catching errors at compile time while maintaining fast compilation speeds.

## Core Design Philosophy

### 1. Architecture Principles
- **Maintainability First**: Design for easy extension and modification by new developers
- **Clean Separation**: Clear boundaries between compilation phases (frontend → semantic → backend)
- **Modular Design**: Each component should be independently testable and replaceable
- **Developer-Friendly**: New contributors should be able to understand and extend features with ease

### 2. Compilation Pipeline
The compiler follows a multi-stage pipeline:
```
Source Code → Lexer → Parser → AST → Collector → Resolver → TypeChecker → CodeGen
```

**Frontend (Priority: High)**
- `lexer/`: Tokenization and lexical analysis
- `parser/`: Syntax parsing and AST generation
- `ast/`: Abstract syntax tree definitions

**Semantic Analysis (Priority: High)**
- `collector/`: Symbol collection and scope building
- `resolver/`: Symbol resolution and reference binding  
- `typecheck/`: Type checking and validation

**Backend (Priority: Low - Future Focus)**
- `codegen/`: Assembly/IR generation (currently x86-64)

### 3. Error Handling Strategy
- **Comprehensive Collection**: Collect ALL errors, warnings, and diagnostics
- **Smart Recovery**: The error system determines when compilation should stop
- **No Manual Stopping**: Let the error reporting system handle flow control
- **Rich Diagnostics**: Provide clear, actionable error messages with source location

### 4. Type System Design
- **Static Typing**: Strong type checking at compile time
- **Type Inference**: Reduce verbosity while maintaining safety
- **Core Features First**: Focus on basic types, structs, arrays, functions
- **Future Advanced Features**: Generics are planned but not current priority

## Development Guidelines

### Language Feature Priorities

**Core Features (Implement First)**
- Variables, arrays, structs
- Functions and methods
- Conditionals and control flow
- Interfaces (core feature)
- Error handling (core feature)
- Module system and imports

**Advanced Features (Future)**
- Generics
- Advanced optimizations
- Multiple backend targets

### Testing Standards
- **Unit Tests Primary**: Focus on comprehensive unit testing
- **Test Coverage**: Each package should have corresponding `*_test.go` files
- **Test Structure**: Use table-driven tests where appropriate
- **Golden Files**: For parser and semantic analysis testing
- **Integration Tests**: For end-to-end compilation workflows

### Performance Goals
- **Fast Compilation**: Prioritize compilation speed over complex optimizations
- **Compile-Time Safety**: Catch errors at compile time, minimize runtime errors
- **Memory Efficiency**: Efficient AST and symbol table representations

### Module System Design
- `fer.ret`: Project root indicator and configuration
- Remote dependencies: GitHub repos stored in config
- Local imports: Relative path-based, no config needed
- Cache management: `.ferret` directory

## Coding Standards

### Error Handling
```go
// Use the report system, don't panic in normal operation
if err != nil {
    ctx.Reports.AddError(location, "descriptive error message")
    return // Let error system decide if compilation continues
}

// Only panic for programming errors or invalid states
if ctx == nil {
    panic("Cannot create parser: Compiler context is nil")
}
```

### Context Management
```go
// Always pass CompilerContext through the pipeline
func NewParser(filePath string, ctx *ctx.CompilerContext, debug bool) *Parser {
    // Validate inputs
    if ctx == nil || filePath == "" {
        panic("Invalid parameters")
    }
    // Use context for error reporting, symbol tables, etc.
}
```

### AST Node Design
```go
// All AST nodes should implement common interfaces
type Node interface {
	INode()
	Loc() *source.Location
}

// Use composition for shared functionality
type BaseNode struct {
    Location *source.Location
}
func (n *BaseNode) INode() {} // Implement Node interface
func (n *BaseNode) Loc() *source.Location {
    return n.Location
}

```

### Testing Patterns
```go
func TestParserFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected interface{}
        wantErr  bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Ferret Language Syntax
#### Variable Declaration
```ferret
let x := 10; // type inferred
let y: i32 = 20; // explicit type declaration
let z: str; // string type
```
#### Array
```ferret
let arr: []i32 = [1, 2, 3]; // array of integers
let arr2: [3]i32 = [1, 2, 3]; // fixed-size array not supported yet
```

#### Types
```ferret
// any types can be declared
type MyType OtherType;

type MyStruct struct {
    field1: i32,
    field2: str
};

type MyInterface interface {
    method1(param: i32) -> str;
    method2() -> void;
}

// struct literals
let myStruct = @MyStruct{field1: 10, field2: "hello"}; // Literals has @ prefix
```

## Development Workflow

### Before Starting Work
1. Run `scripts/test.*` to ensure current functionality works
2. Check existing tests for similar functionality
3. Consider impact on compilation pipeline stages

### During Development
1. Write unit tests first (TDD approach when possible)
2. Use debug mode for testing: `ferret filename.fer --debug`
3. Test with sample files in `app/` directory
4. Run `scripts/ci-check.*` before committing

### Code Review Standards
- All new features must have tests
- Error messages should be clear and actionable
- Consider impact on compilation performance
- Maintain backward compatibility in AST structures
- Document any new compiler phases or major changes

### Code Testing while developing
- To compile or build the compiler, use:
```bash
./build.sh        # Unix
.\build.bat       # Windows
```
From `scripts/` directory. (Mandatory)

Then go to the `app/` directory and run:
```bash
ferret run --debug
```
Make sure you add the `bin` directory to your `PATH` environment variable to run the `ferret` command from anywhere.

### Available Scripts
```bash
# Build compiler
./scripts/build.sh        # Unix
.\scripts\build.bat       # Windows

# Run tests with formatting
./scripts/test.sh         # Unix
.\scripts\test.bat        # Windows

# Format code
./scripts/fmt.sh          # Unix
.\scripts\fmt.bat         # Windows

# Quick development test
./scripts/run.sh          # Unix
.\scripts\run.bat         # Windows

# Build the LSP server
./scripts/lsp.sh     # Unix
.\scripts\lsp.bat    # Windows

# Build the LSP server + package the vscode extension
./scripts/pack.sh     # Unix
.\scripts\pack.bat    # Windows

# Full CI validation
./scripts/ci-check.sh     # Unix
.\scripts\ci-check.bat    # Windows
```

## Common Patterns

### Adding New Language Features
1. Update lexer for new tokens (if needed)
2. Extend parser for new syntax
3. Add AST node types
4. Update semantic analysis phases
5. Add comprehensive tests
6. Update language documentation

### Extending Error Reporting
```go
// Add new error types in report/constants.go
// Use consistent error formatting
ctx.Reports.AddError(location, fmt.Sprintf("Expected %s, found %s", expected, actual))
```

### Working with Symbol Tables
```go
// Use compiler context for symbol resolution
symbol := ctx.Symbols.Lookup(identifier)
if symbol == nil {
    ctx.Reports.AddError(location, fmt.Sprintf("Undefined identifier: %s", identifier))
}
```

This instruction file should be updated as the compiler evolves and new patterns emerge.