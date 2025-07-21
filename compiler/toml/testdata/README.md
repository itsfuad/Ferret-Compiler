# Test Data Files

This directory contains TOML test files used by the parser test suite.

## Test Files

### Basic Functionality
- **`basic.toml`** - Simple key-value pairs with different data types
- **`empty.toml`** - Empty file to test parsing of empty inputs
- **`only_comments.toml`** - File containing only comments

### Section Handling
- **`with_sections.toml`** - TOML file with multiple sections
- **`with_comments.toml`** - Sections and comments mixed together

### Value Type Testing
- **`mixed_types.toml`** - All supported value types (string, int, float, bool, raw)

### Edge Cases
- **`invalid.toml`** - Invalid TOML syntax for error testing
- **`multiple_equals.toml`** - Values containing multiple equals signs (URLs, connection strings)
- **`whitespace_handling.toml`** - Testing various whitespace scenarios

### Complex Examples
- **`complex.toml`** - Real-world configuration example with multiple sections
- **`benchmark.toml`** - Larger file used for performance benchmarking

## Usage

These files are used by `parser_test.go` via the `getTestDataPath()` helper function.
Each test case references a specific file to ensure consistent and manageable test data.

## Benefits of File-Based Tests

1. **Maintainability**: Easier to edit and version control test data
2. **Readability**: Clear separation between test logic and test data
3. **Reusability**: Test files can be used by multiple test functions
4. **Real-world**: Files represent actual use cases better than inline strings
5. **Debugging**: Easy to inspect and manually validate test data
