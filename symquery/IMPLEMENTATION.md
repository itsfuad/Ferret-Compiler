# Symbol Query Server - Implementation Summary

## Overview

The Symbol Query Server (`symquery`) is a standalone program that compiles Ferret projects up to the typecheck phase and provides an interactive or programmatic interface for querying symbol information. This tool is designed to support LSP (Language Server Protocol) implementations and other development tools that need access to symbol information.

## Architecture

### Components

1. **symquery/main.go** - Main program with interactive and JSON modes
2. **compiler/cmd/symbol_api.go** - Public API for accessing compiler internals
3. **Scripts** - Build and run scripts (symquery.sh, symquery.bat)

### Design Decisions

#### Public API Layer
Instead of directly accessing internal compiler packages (which would violate Go's internal package restrictions), we created a public API layer (`symbol_api.go`) that:
- Exposes only the necessary functionality
- Provides type-safe access to symbol information
- Maintains encapsulation of internal compiler details
- Converts internal types to public DTO (Data Transfer Object) types

#### Why Not Use Internal Packages Directly?
Go's `internal` package convention prevents external packages from importing them. This is a deliberate design choice to:
- Enforce API boundaries
- Prevent tight coupling
- Allow internal refactoring without breaking external tools
- Maintain clean separation of concerns

### Data Flow

```
symquery.exe (external tool)
    ↓
cmd.CompileForSymbolQuery() (public API)
    ↓
cmd.Compile() → CompilerContext (internal)
    ↓
cmd.SymbolQueryAPI (public wrapper)
    ↓
symquery queries (interactive/JSON)
```

## Features

### 1. Compilation
- Compiles Ferret projects through all phases up to typecheck
- Keeps compiler context in memory for fast queries
- Handles errors gracefully (partial symbol table available even with errors)

### 2. Interactive Mode
- REPL-style interface
- Human-readable colored output
- Easy command syntax
- Built-in help system

### 3. JSON Mode
- Line-delimited JSON input/output
- Suitable for programmatic access
- Designed for LSP server integration
- Consistent response format

### 4. Symbol Information
For each symbol, provides:
- **Name**: Symbol identifier
- **Kind**: variable, constant, type, function, method, struct, field
- **Type**: Type information as string
- **Location**: File path, line number, column number
- **Scope**: Module/file where symbol is defined
- **Exported**: Whether symbol is exported (capitalized)

### 5. Query Commands
- `query <symbol>` - Find specific symbol across all modules
- `list` - List all symbols from all modules
- `modules` - List all loaded modules with metadata
- `stats` - Compilation statistics (module count, symbol count, errors)
- `help` - Command documentation

## Implementation Details

### Symbol Location Handling

The Ferret compiler uses a hierarchical location system:
- **Position**: Line, Column, Index in source
- **Location**: Start Position + End Position (a span)
- **Symbol**: Contains Location reference

The file path is not stored in Position/Location but is inferred from:
- Module path (key in Modules map)
- Module AST's FullPath field

This design keeps Position/Location lightweight and allows the same Location to be reused across references.

### Module Information

Modules are keyed by their import path, not file path:
- Key: `"app/cmd/main.fer"` (import path)
- Value: Module with AST (contains actual file path)

This distinction is important for:
- Import resolution
- Symbol namespacing
- Cross-module references

### Type System Integration

The tool leverages Ferret's semantic type system:
- Uses `stype.Type.String()` for human-readable type representation
- Handles built-in types (i32, f64, str, bool, void)
- Supports user-defined types (structs, type aliases)
- Function signatures with parameters and return types

## Use Cases

### 1. LSP Server Integration
```go
// Start symquery in JSON mode
process := startProcess("symquery", project, "--json")

// Query symbol at cursor position
send(process, `{"command":"query","symbol":"myFunction"}`)
response := readJSON(process)

// Use response for "Go to Definition", "Hover", etc.
showDefinition(response.Data[0].Location)
```

### 2. IDE Plugin Development
- Symbol search and navigation
- Type information on hover
- Auto-completion suggestions
- Code navigation (go to definition, find references)

### 3. Documentation Generation
- Extract all exported symbols
- Generate API documentation
- Create symbol index

### 4. Code Analysis Tools
- Find unused symbols
- Analyze symbol dependencies
- Track API evolution

## Future Enhancements

### Planned Features
1. **Position-based queries**: Get symbol at specific line:column
2. **Find references**: All uses of a symbol
3. **Scope-aware lookup**: Context-sensitive symbol resolution
4. **Watch mode**: Auto-recompile on file changes
5. **Incremental compilation**: Update only changed modules
6. **Symbol hierarchy**: Parent types, implementations, overrides
7. **Documentation comments**: Extract and return doc comments
8. **Rename support**: Validate and preview rename operations

### Performance Optimizations
1. **Caching**: Cache compiled modules for faster reloading
2. **Lazy loading**: Load modules on-demand
3. **Parallel compilation**: Compile independent modules concurrently
4. **Index building**: Pre-build symbol index for fast lookup

### Enhanced LSP Support
1. **Workspace symbols**: Cross-file symbol search
2. **Document symbols**: Hierarchical symbol tree for a file
3. **Code lens**: Inline reference counts, implementations
4. **Call hierarchy**: Function call graphs
5. **Type hierarchy**: Type inheritance and implementations

## Integration Example

### Python LSP Server
```python
import subprocess
import json

class SymbolQueryClient:
    def __init__(self, project_path):
        self.process = subprocess.Popen(
            ['symquery', project_path, '--json'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            text=True
        )
    
    def query_symbol(self, name):
        query = {"command": "query", "symbol": name}
        self.process.stdin.write(json.dumps(query) + '\n')
        self.process.stdin.flush()
        response = json.loads(self.process.stdout.readline())
        return response
    
    def get_statistics(self):
        query = {"command": "stats"}
        self.process.stdin.write(json.dumps(query) + '\n')
        self.process.stdin.flush()
        return json.loads(self.process.stdout.readline())

# Usage
client = SymbolQueryClient('/path/to/project')
result = client.query_symbol('main')
print(result['data'][0]['location'])  # File, line, column
```

### VS Code Extension
```typescript
import { spawn } from 'child_process';

class SymbolQueryServer {
    private process: any;
    
    constructor(projectPath: string) {
        this.process = spawn('symquery', [projectPath, '--json']);
    }
    
    async querySymbol(symbol: string): Promise<any> {
        return new Promise((resolve) => {
            const query = { command: 'query', symbol };
            this.process.stdin.write(JSON.stringify(query) + '\n');
            
            this.process.stdout.once('data', (data: Buffer) => {
                const response = JSON.parse(data.toString());
                resolve(response);
            });
        });
    }
}
```

## Testing

### Manual Testing
```bash
# Build
cd symquery
go build

# Interactive mode
./symquery ../app

# Test commands
symquery> stats
symquery> query main
symquery> list
symquery> modules
```

### JSON Mode Testing
```bash
echo '{"command":"stats"}' | ./symquery ../app --json
echo '{"command":"query","symbol":"main"}' | ./symquery ../app --json
```

### Integration Testing
Create test scripts that:
1. Start symquery in JSON mode
2. Send various queries
3. Validate response format
4. Check symbol information accuracy
5. Test error handling

## Conclusion

The Symbol Query Server provides a robust foundation for LSP and tooling support in the Ferret compiler ecosystem. By separating the query interface from the compiler internals, it allows for:
- Clean API boundaries
- Easy integration with external tools
- Future extensibility without breaking changes
- Support for multiple use cases (IDE, CLI, CI/CD)

The current implementation focuses on core functionality (up to typecheck), which is sufficient for most LSP features like go-to-definition, hover information, and symbol search. Future enhancements will expand capabilities while maintaining backward compatibility.
