# Symbol Query Server (symquery)

A standalone program that compiles Ferret projects up to the typecheck phase and provides an interactive query interface for symbol information. This tool is designed to support LSP (Language Server Protocol) implementations and other development tools that need access to symbol information.

## Overview

The Symbol Query Server compiles your Ferret project and keeps the compiler context in memory, allowing you to query for symbol information without recompiling. This makes it efficient for tools that need to frequently access symbol information.

## Features

- **Compilation to Typecheck**: Compiles the project through parsing, symbol collection, resolution, and typechecking
- **Interactive Mode**: Query symbols through a REPL-style interface
- **JSON Mode**: Programmatic access via JSON-RPC for integration with other tools
- **Symbol Information**: Get detailed information about variables, functions, types, etc.
- **Module Inspection**: View all loaded modules and their symbols
- **Statistics**: Get compilation statistics and error counts

## Building

From the `symquery` directory:

```bash
go build -o symquery.exe
```

Or from the project root:

```bash
cd symquery
go build
```

## Usage

### Interactive Mode (Default)

Start the query server in interactive mode:

```bash
./symquery <project-root>
```

Example:
```bash
./symquery ../app
```

With debug output:
```bash
./symquery ../app --debug
```

### JSON Mode

For programmatic access (suitable for LSP integration):

```bash
./symquery <project-root> --json
```

## Commands

### Interactive Mode Commands

Once in interactive mode, you can use the following commands:

#### `query <symbol>` or `find <symbol>`
Find information about a specific symbol by name.

```
symquery> query myVariable
✅ Success:
[
  {
    "name": "myVariable",
    "kind": "variable",
    "type": "i32",
    "location": {
      "file": "/path/to/file.fer",
      "line": 42,
      "column": 5
    },
    "scope": "app/cmd/main.fer",
    "exported": false
  }
]
```

#### `list`
List all symbols from all modules.

```
symquery> list
✅ Success:
{
  "builtin": [
    {
      "name": "i32",
      "kind": "type",
      "type": "i32",
      "scope": "builtin",
      "exported": false
    },
    ...
  ],
  "app/cmd/main.fer": [
    {
      "name": "main",
      "kind": "function",
      "type": "func() -> void",
      "location": {...},
      "scope": "app/cmd/main.fer",
      "exported": false
    }
  ]
}
```

#### `modules`
List all loaded modules with basic information.

```
symquery> modules
✅ Success:
{
  "app/cmd/main.fer": {
    "full_path": "/absolute/path/to/app/cmd/main.fer",
    "symbol_count": 15
  },
  ...
}
```

#### `stats` or `statistics`
Show compilation statistics.

```
symquery> stats
✅ Success:
{
  "modules": 5,
  "total_symbols": 127,
  "builtin_symbols": 12,
  "has_errors": false,
  "error_count": 0
}
```

#### `help`
Show available commands.

```
symquery> help
✅ Success:
{
  "query <symbol>": "Find information about a specific symbol",
  "list": "List all symbols from all modules",
  "modules": "List all loaded modules",
  "stats": "Show compilation statistics",
  "help": "Show this help message",
  "exit": "Exit the query server"
}
```

#### `exit` or `quit`
Exit the query server.

```
symquery> exit
👋 Goodbye!
```

## JSON Mode Protocol

In JSON mode, send queries as JSON objects on stdin, one per line. Each query should follow this format:

### Request Format

```json
{
  "command": "query",
  "symbol": "myVariable"
}
```

Available commands:
- `query` / `find` - requires `symbol` field
- `list` - lists all symbols
- `modules` - lists all modules
- `stats` / `statistics` - shows statistics
- `help` - shows help information

### Response Format

Success response:
```json
{
  "success": true,
  "data": { ... }
}
```

Error response:
```json
{
  "success": false,
  "error": "error message"
}
```

### Example JSON Mode Session

Input:
```json
{"command": "query", "symbol": "main"}
```

Output:
```json
{
  "success": true,
  "data": [
    {
      "name": "main",
      "kind": "function",
      "type": "func() -> void",
      "location": {
        "file": "/path/to/main.fer",
        "line": 10,
        "column": 1
      },
      "scope": "app/cmd/main.fer",
      "exported": false
    }
  ]
}
```

## Symbol Information Fields

Each symbol returned includes:

- **name**: Symbol name
- **kind**: Symbol kind (`variable`, `constant`, `type`, `function`, `method`, `struct`, `field`)
- **type**: Type information as a string
- **location**: Source location (file, line, column) - if available
- **scope**: Module/scope where the symbol is defined
- **exported**: Whether the symbol is exported (starts with uppercase)

## Use Cases

### For LSP Implementation

The JSON mode is designed for LSP integration:

1. Start the server in JSON mode
2. Send symbol queries as the user types/navigates
3. Get instant symbol information without recompiling
4. Use for features like:
   - Go to definition
   - Find references
   - Hover information
   - Symbol search
   - Autocomplete

### For Development Tools

Interactive mode is useful for:
- Debugging compiler issues
- Understanding project structure
- Exploring symbol tables
- Verifying type checking results

## Error Handling

The server will compile the project even if there are errors, making partial symbol information available. Check the `has_errors` field in statistics to see if compilation had issues.

```
symquery> stats
✅ Success:
{
  "has_errors": true,
  "error_count": 3,
  ...
}
```

## Future Enhancements

Planned features for LSP support:
- Find all references to a symbol
- Rename symbol across project
- Symbol hierarchy (parent types, implementations)
- Scope-aware symbol lookup
- Position-based queries (get symbol at line:col)
- Watch mode (recompile on file changes)
- Incremental compilation support

## Integration Example

### Python Client Example

```python
import subprocess
import json

# Start the server
proc = subprocess.Popen(
    ['./symquery', 'path/to/project', '--json'],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    text=True
)

# Query for a symbol
query = {"command": "query", "symbol": "myFunction"}
proc.stdin.write(json.dumps(query) + '\n')
proc.stdin.flush()

# Read response
response = json.loads(proc.stdout.readline())
if response['success']:
    print(response['data'])
else:
    print(f"Error: {response['error']}")
```

### Node.js Client Example

```javascript
const { spawn } = require('child_process');

const server = spawn('./symquery', ['path/to/project', '--json']);

// Send query
const query = { command: 'query', symbol: 'myFunction' };
server.stdin.write(JSON.stringify(query) + '\n');

// Read response
server.stdout.on('data', (data) => {
  const response = JSON.parse(data.toString());
  if (response.success) {
    console.log(response.data);
  } else {
    console.error(response.error);
  }
});
```

## Notes

- The server compiles the project once at startup
- Symbol information is based on the state at compilation time
- For updated information, restart the server (or use future watch mode)
- The server currently runs up to typecheck phase (codegen not included)
