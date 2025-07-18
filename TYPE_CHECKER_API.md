# Cleaned Type Checker API

This document describes the clean, focused type checker API that follows core rules.

## Core Functions

### 1. `GetTypeFromAST(astType ast.DataType) semantic.Type`
Converts an AST DataType to a semantic Type.
- Handles all AST type nodes (primitives, arrays, structs, functions, user types)
- Returns nil for invalid AST types
- Uses the existing `semantic.ASTToSemanticType` function

### 2. `InferTypeFromExpression(r *analyzer.AnalyzerNode, expr ast.Expression, cm *ctx.Module) semantic.Type`
Infers the semantic type from an AST expression.
- Handles literals (string, int, float, bool, byte)
- Handles complex expressions (identifiers, binary ops, array literals, indexing)
- Returns nil for unknown expression types
- Includes error reporting for invalid operations

### 3. `IsAssignableFrom(target, source semantic.Type) bool`
Checks if a value of type 'source' can be assigned to 'target'.
- Handles exact type matches
- Resolves user type aliases
- Supports numeric promotions (int8 -> int16 -> int32 -> int64, int -> float)
- Supports array compatibility (element type assignability)
- Supports function compatibility (contravariant parameters, covariant returns)
- Supports struct compatibility (structural typing)

## Helper Functions

### Type Checking Utilities
- `CanImplicitlyConvert(target, source semantic.Type) bool` - alias for `IsAssignableFrom`
- `CanExplicitlyConvert(target, source semantic.Type) bool` - allows all numeric conversions
- `resolveUserType(t semantic.Type) semantic.Type` - resolves type aliases

### Expression Type Inference Helpers
- `inferIdentifierType(e *ast.IdentifierExpr, cm *ctx.Module) semantic.Type`
- `inferBinaryExprType(r *analyzer.AnalyzerNode, e *ast.BinaryExpr, cm *ctx.Module) semantic.Type`
- `inferArrayLiteralType(r *analyzer.AnalyzerNode, e *ast.ArrayLiteralExpr, cm *ctx.Module) semantic.Type`
- `inferIndexableType(r *analyzer.AnalyzerNode, e *ast.IndexableExpr, cm *ctx.Module) semantic.Type`

### Binary Operation Helpers
- `getBinaryOperationResultType(operator string, left, right semantic.Type) semantic.Type`
- `getArithmeticResultType(operator string, left, right semantic.Type) semantic.Type`
- `getComparisonResultType(left, right semantic.Type) semantic.Type`
- `getLogicalResultType(left, right semantic.Type) semantic.Type`
- `getBitwiseResultType(left, right semantic.Type) semantic.Type`
- `getCommonNumericType(left, right semantic.Type) semantic.Type`

### Type Compatibility Helpers
- `isNumericPromotion(target, source semantic.Type) bool`
- `isArrayCompatible(target, source semantic.Type) bool`
- `isFunctionCompatible(target, source semantic.Type) bool`
- `isStructCompatible(target, source semantic.Type) bool`

### Type Classification Utilities
- `isStringType(t semantic.Type) bool`
- `isBoolType(t semantic.Type) bool`
- `isNumericType(t semantic.Type) bool`
- `isIntegerType(t semantic.Type) bool`
- `isNumericTypeName(typeName types.TYPE_NAME) bool`
- `isIntegerTypeName(typeName types.TYPE_NAME) bool`

## Type System Rules

### Numeric Promotions
```
int8 -> int16 -> int32 -> int64
uint8/byte -> uint16 -> uint32 -> uint64
int8/int16 -> float32
int8/int16/int32/uint8/uint16/uint32/byte -> float64
float32 -> float64
```

### Assignment Rules
1. Exact type match is always allowed
2. User type aliases are resolved to their underlying types
3. Numeric promotions are allowed (smaller -> larger)
4. Array assignment requires element type compatibility
5. Function assignment requires parameter contravariance and return covariance
6. Struct assignment requires structural compatibility (all required fields present)

### Operation Rules
1. Arithmetic operations return the common numeric type of operands
2. String concatenation only allows string + string
3. Comparison operations return bool if operands are comparable
4. Logical operations require bool operands and return bool
5. Bitwise operations require integer operands and return common integer type
6. Array indexing requires integer index and returns element type

## Usage Examples

```go
// Type conversion
astType := &ast.IntType{TypeName: types.INT32}
semanticType := GetTypeFromAST(astType)

// Type inference
exprType := InferTypeFromExpression(analyzer, expression, module)

// Assignability check
if IsAssignableFrom(targetType, sourceType) {
    // Assignment is valid
}

// Conversion checks
if CanImplicitlyConvert(target, source) {
    // Implicit conversion allowed
}
if CanExplicitlyConvert(target, source) {
    // Explicit cast allowed
}
```

This clean API eliminates ambiguity and provides a clear foundation for type checking operations.
