# Symbol Query Server Test Script

This script demonstrates the Symbol Query Server functionality.

## Test Commands

### 1. Get help
```
help
```

### 2. Query a specific symbol
```
query add
```

### 3. List all symbols
```
list
```

### 4. List all modules
```
modules
```

### 5. Get compilation statistics
```
stats
```

### 6. Exit
```
exit
```

## Example Session

```
symquery> help
✅ Success:
{
  "exit": "Exit the query server",
  "help": "Show this help message",
  "list": "List all symbols from all modules",
  "modules": "List all loaded modules",
  "query <symbol>": "Find information about a specific symbol",
  "stats": "Show compilation statistics"
}

symquery> stats
✅ Success:
{
  "builtin_symbols": 24,
  "error_count": 1,
  "has_errors": true,
  "modules": 1,
  "total_symbols": 30
}

symquery> query add
✅ Success:
[
  {
    "name": "add",
    "kind": "function",
    "type": "func(i32, i32) -> i32",
    "location": {
      "file": "D:/dev/Golang/Ferret-Compiler/app/cmd/test.fer",
      "line": 8,
      "column": 1
    },
    "scope": "D:/dev/Golang/Ferret-Compiler/app/cmd/test.fer",
    "exported": false
  }
]

symquery> exit
👋 Goodbye!
```

## JSON Mode Usage

For programmatic access:

```bash
# Start in JSON mode
.\symquery.exe ..\app --json

# Send JSON queries (one per line)
{"command": "stats"}
{"command": "query", "symbol": "add"}
{"command": "list"}
{"command": "modules"}
```

Response format:
```json
{
  "success": true,
  "data": { ... }
}
```

Or error:
```json
{
  "success": false,
  "error": "error message"
}
```
