# Scripts Directory

This directory contains build and development scripts for the Ferret Programming Language project.

## Available Scripts

All scripts are available in both Windows batch (`.bat`) and Unix shell (`.sh`) formats.

### Core Development Scripts

#### `build.bat` / `build.sh`
Builds the Ferret compiler with optimizations.
- **Purpose**: Creates an optimized binary of the compiler
- **Output**: `compiler/bin/ferret.exe` (Windows) or `compiler/bin/ferret` (Unix)
- **Usage**: 
  ```bash
  # Windows Command Prompt/PowerShell
  .\scripts\build.bat
  
  # Linux/macOS/Git Bash
  ./scripts/build.sh
  ```

#### `run.bat` / `run.sh`
Runs the compiler with a sample file for quick testing.
- **Purpose**: Quick development testing
- **Target**: Compiles and runs `app/cmd/main.fer` with debug output
- **Usage**:
  ```bash
  # Windows Command Prompt/PowerShell
  .\scripts\run.bat
  
  # Linux/macOS/Git Bash
  ./scripts/run.sh
  ```

#### `test.bat` / `test.sh`
Runs the complete test suite with formatted output.
- **Purpose**: Execute all unit and integration tests
- **Features**: Colored output, test statistics, pass/fail reporting
- **Usage**:
  ```bash
  # Windows Command Prompt/PowerShell
  .\scripts\test.bat
  
  # Linux/macOS/Git Bash
  ./scripts/test.sh
  ```

#### `fmt.bat` / `fmt.sh`
Formats all Go code in the project.
- **Purpose**: Code formatting using `go fmt`
- **Scope**: All Go files in the compiler directory
- **Usage**:
  ```bash
  # Windows Command Prompt/PowerShell
  .\scripts\fmt.bat
  
  # Linux/macOS/Git Bash
  ./scripts/fmt.sh
  ```

### Quality Assurance Scripts

#### `ci-check.bat` / `ci-check.sh`
Comprehensive local CI simulation.
- **Purpose**: Run all CI checks locally before pushing
- **Includes**: 
  - Dependency download
  - Code formatting validation
  - Static analysis (`go vet`)
  - Complete test suite
  - Build verification
  - CLI functionality testing
- **Usage**:
  ```bash
  # Windows Command Prompt/PowerShell
  .\scripts\ci-check.bat
  
  # Linux/macOS/Git Bash
  ./scripts/ci-check.sh
  ```

### Extension Scripts

#### `pack.bat` / `pack.sh`
Packages the VS Code language extension.
- **Purpose**: Creates `.vsix` package for VS Code extension
- **Requirements**: `vsce` tool must be installed
- **Usage**:
  ```bash
  # Windows Command Prompt/PowerShell
  .\scripts\pack.bat
  
  # Linux/macOS/Git Bash
  ./scripts/pack.sh
  ```

## Usage Guidelines

### Before Committing
Always run the CI check script to ensure your changes will pass the pipeline:
```bash
# Windows Command Prompt/PowerShell
.\scripts\ci-check.bat

# Linux/macOS/Git Bash
./scripts/ci-check.sh
```

### Development Workflow
1. **Make changes** to the code
2. **Format code**: `./scripts/fmt.sh`
3. **Run tests**: `./scripts/test.sh`
4. **Test compilation**: `./scripts/build.sh`
5. **Quick test**: `./scripts/run.sh`
6. **Final check**: `./scripts/ci-check.sh`

### Cross-Platform Notes
- All scripts maintain the same functionality across platforms
- Paths are adjusted automatically for the operating system
- Shell scripts include enhanced features like colored output
- Windows batch files are optimized for cmd/PowerShell compatibility
- **Git Bash support**: Shell scripts (`.sh`) can be run on Windows using Git Bash
  ```bash
  # Using Git Bash on Windows
  cd /d/dev/Golang/Ferret_Compiler/scripts
  ./ci-check.sh
  ```

## Requirements

### All Scripts
- Go 1.19 or later
- Access to the compiler source code

### Extension Packaging
- Node.js and npm
- Visual Studio Code Extension Manager (`vsce`)
  ```bash
  npm install -g vsce
  ```

## Script Locations
All scripts are located in the `scripts/` directory and expect to be run from that location. They automatically navigate to the appropriate project directories as needed.
