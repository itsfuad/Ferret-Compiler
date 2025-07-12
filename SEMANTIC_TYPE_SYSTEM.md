# Ferret Compiler Type System Architecture

## Overview

The Ferret compiler now uses a **2-tier type system** that separates syntactic representation from semantic analysis:

1. **AST Types** (`ast.DataType`) - Syntax with location information
2. **Semantic Types** (`semantic.Type`) - Lightweight types for analysis

## Architecture Benefits

### Clear Separation of Concerns
- **AST Types**: Represent parsed source code with source location info for error reporting
- **Semantic Types**: Represent the type system for analysis without location dependency

### Performance Improvements
- Symbols store lightweight semantic types instead of heavy AST nodes
- No unnecessary location information overhead in symbol table
- Efficient type comparisons and operations

### Maintainability
- Changes to AST don't affect type checker
- Clean interfaces between compiler phases
- Focused responsibilities for each type system

## Type System Components

### 1. Semantic Type Interface (`semantic.Type`)

```go
type Type interface {
    TypeName() types.TYPE_NAME
    String() string
    Equals(other Type) bool
}
```

### 2. Semantic Type Implementations

- **`PrimitiveType`**: Built-in types (int, string, bool, etc.)
- **`UserType`**: User-defined types and type aliases
- **`StructType`**: Struct types with named fields
- **`ArrayType`**: Array types with element types
- **`FunctionType`**: Function types with parameters and return types

### 3. Type Conversion Layer (`semantic/typeconv.go`)

Provides utilities to convert between AST types and semantic types:

```go
// Convert AST DataType to semantic Type
func ASTToSemanticType(astType ast.DataType) Type

// Create semantic types directly
func CreatePrimitiveType(typeName types.TYPE_NAME) Type
func CreateUserType(typeName types.TYPE_NAME) Type
func CreateStructType(typeName types.TYPE_NAME, fields map[string]Type) Type
```

### 4. Updated Symbol System (`semantic/symbols.go`)

```go
type Symbol struct {
    Name     string
    Kind     SymbolKind
    Type     semantic.Type      // Now uses semantic types
    Location *source.Location   // Optional, only when needed for error reporting
}
```

### 5. Type Checker (`semantic/typecheck/typechecker.go`)

The type checker follows the same pattern as the resolver - using analyzer node functions instead of a struct:

```go
// Main entry point
func CheckProgram(r *analyzer.AnalyzerNode)

// Internal functions
func checkNode(r *analyzer.AnalyzerNode, node ast.Node)
func checkVarDecl(r *analyzer.AnalyzerNode, stmt *ast.VarDeclStmt)
func checkAssignment(r *analyzer.AnalyzerNode, stmt *ast.AssignmentStmt)
func inferExpressionType(r *analyzer.AnalyzerNode, expr ast.Expression) semantic.Type
```

- Performs semantic type analysis
- Validates type compatibility
- Infers expression types
- Reports type errors with proper source locations

## Compiler Pipeline Integration

### Phase 1: Lexing & Parsing
- Creates AST with `ast.DataType` nodes
- Preserves source location information

### Phase 2: Resolver
- Converts AST types to semantic types using `ASTToSemanticType()`
- Stores semantic types in symbol table
- Handles name resolution

### Phase 3: Type Checker
- Works with semantic types from symbol table
- Performs type inference and validation
- Reports errors using stored location information when needed

## Usage Examples

### Creating Semantic Types

```go
// Primitive types
intType := semantic.CreatePrimitiveType(types.INT32)
stringType := semantic.CreatePrimitiveType(types.STRING)

// User-defined types
userType := semantic.CreateUserType(types.TYPE_NAME("MyType"))

// Struct types
fields := map[string]semantic.Type{
    "name": stringType,
    "age":  intType,
}
structType := semantic.CreateStructType(types.TYPE_NAME("Person"), fields)
```

### Converting AST to Semantic Types

```go
// During resolver phase
astType := varDecl.ExplicitType  // ast.DataType
semanticType := semantic.ASTToSemanticType(astType)  // semantic.Type

// Store in symbol
sym := semantic.NewSymbol(name, kind, semanticType)
symbolTable.Declare(name, sym)
```

### Type Checking

```go
// During type checker phase - using analyzer node directly
func CheckProgram(r *analyzer.AnalyzerNode) {
    for _, node := range r.Program.Nodes {
        checkNode(r, node)
    }
}

leftType := inferExpressionType(r, leftExpr)
rightType := inferExpressionType(r, rightExpr)

if !semantic.IsAssignableFrom(leftType, rightType) {
    // Report type mismatch error
}
```

## Current Status

✅ **Fully Implemented:**
- Complete semantic type system with all interfaces and implementations
- Comprehensive type conversion utilities with advanced compatibility rules
- Updated symbol system using semantic types
- Updated resolver using semantic types throughout
- Complete type checker with full expression type inference and binary operation support
- Updated prelude with semantic types
- **Advanced Type Compatibility Features:**
  - Numeric type promotions (e.g., i8 → i32 → i64, int → float)
  - Array type compatibility with element type checking
  - Function type compatibility with contravariant parameters and covariant returns
  - Structural typing for struct compatibility
  - Binary operation type inference with proper operator semantics
  - Explicit and implicit type conversion checking

✅ **Working Features:**
- Primitive type handling with full promotion rules
- User-defined type handling with alias resolution
- Variable declaration type checking with initializer compatibility
- Assignment type checking with comprehensive compatibility rules
- Field access validation with struct type checking
- Binary expressions with proper type inference and operator validation
- All existing tests pass with enhanced type safety

🔄 **Future Enhancements:**
- Interface type checking and implementation validation
- Generic type support with type parameters
- More sophisticated error messages with type suggestions and fixes
- Advanced control flow analysis for unreachable code detection
- Type inference improvements for complex expressions

## Migration Notes

### For Existing Code:
- Symbol creation now uses `semantic.NewSymbol()` instead of direct struct literal
- Type information access uses `symbol.Type.TypeName()` instead of `symbol.Type.Type()`
- Type comparisons use `type1.Equals(type2)` instead of direct comparison

### For New Features:
- Use semantic types directly for type analysis
- Convert to AST types only when source location is needed
- Prefer lightweight semantic operations over AST traversal
- Follow analyzer node function pattern (like resolver) instead of creating structs

This architecture provides a solid foundation for the Ferret compiler's type system, enabling sophisticated type checking while maintaining clean separation between syntax and semantics.
