# Ferret Programming Language
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
└── docs/             # Documentation and examples
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

## License
This project is licensed under the Mozilla Public License 2.0 - see the LICENSE file for details.
