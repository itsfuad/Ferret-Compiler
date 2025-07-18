# Regex Patterns for Report Syntax Migration

This file contains regex patterns to migrate from the old `Reports.Add(...).SetLevel(...)` syntax to the new direct methods.

## VS Code Regex Patterns

Use these patterns in VS Code's Find and Replace (Ctrl+H) with Regex enabled:

### Pattern 1: SEMANTIC_ERROR
```
Find: (\.Reports)\.Add\(([^)]+)\)\.SetLevel\(report\.SEMANTIC_ERROR\)
Replace: $1.AddSemanticError($2)
```

### Pattern 2: CRITICAL_ERROR
```
Find: (\.Reports)\.Add\(([^)]+)\)\.SetLevel\(report\.CRITICAL_ERROR\)
Replace: $1.AddCriticalError($2)
```

### Pattern 3: SYNTAX_ERROR
```
Find: (\.Reports)\.Add\(([^)]+)\)\.SetLevel\(report\.SYNTAX_ERROR\)
Replace: $1.AddSyntaxError($2)
```

### Pattern 4: NORMAL_ERROR
```
Find: (\.Reports)\.Add\(([^)]+)\)\.SetLevel\(report\.NORMAL_ERROR\)
Replace: $1.AddError($2)
```

### Pattern 5: WARNING
```
Find: (\.Reports)\.Add\(([^)]+)\)\.SetLevel\(report\.WARNING\)
Replace: $1.AddWarning($2)
```

### Pattern 6: INFO
```
Find: (\.Reports)\.Add\(([^)]+)\)\.SetLevel\(report\.INFO\)
Replace: $1.AddInfo($2)
```

## Advanced Pattern (Single Pattern for All Types)

If you want to use a single pattern that handles all types:

```
Find: (\.Reports)\.Add\(([^)]+)\)\.SetLevel\(report\.([A-Z_]+)\)
Replace: $1.Add$3($2)
```

But you'll need to manually fix the method names:
- `SEMANTIC_ERROR` → `AddSemanticError`
- `CRITICAL_ERROR` → `AddCriticalError`  
- `SYNTAX_ERROR` → `AddSyntaxError`
- `NORMAL_ERROR` → `AddError`
- `WARNING` → `AddWarning`
- `INFO` → `AddInfo`

## How to Use

1. Open VS Code
2. Press `Ctrl+Shift+F` (Find in Files)
3. Enable regex mode (.*) button
4. Use the search patterns to find all instances
5. Use `Ctrl+H` (Replace) with regex enabled to replace them

## Files to Update

Based on our search, you need to update these files:
- `compiler/internal/semantic/resolver/resolver.go`
- `compiler/internal/semantic/resolver/declarations.go`
- `compiler/internal/semantic/collector/collector.go`
- `compiler/internal/frontend/parser/variableDecl.go`
- `compiler/internal/frontend/parser/variableAssign.go`

## Note

The typecheck package files have already been updated in this session:
- ✅ `compiler/internal/semantic/typecheck/utils.go`
- ✅ `compiler/internal/semantic/typecheck/declarations.go`
- ✅ `compiler/internal/semantic/typecheck/typecheck.go`

## Example Transformation

Before:
```go
r.Ctx.Reports.Add(r.Program.FullPath, variable.ExplicitType.Loc(), "Explicit type does not match initializer type", report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
```

After:
```go
r.Ctx.Reports.AddSemanticError(r.Program.FullPath, variable.ExplicitType.Loc(), "Explicit type does not match initializer type", report.TYPECHECK_PHASE)
```

This eliminates the need for the `SetLevel()` call and makes the code more readable and direct.
