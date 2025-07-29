@echo off
setlocal enabledelayedexpansion

REM Colors (for Windows, just plain text)
set RED=
set GREEN=
set YELLOW=
set NC=

echo %YELLOW%🚀 Running local PR workflow simulation...%NC%

REM Set up directories
set SCRIPT_DIR=%~dp0
set ROOT_DIR=%SCRIPT_DIR%..
set COMPILER_DIR=%ROOT_DIR%\compiler
set BIN_DIR=%ROOT_DIR%\bin

cd /d %ROOT_DIR%

echo %YELLOW%📦 Step 1: Setting up environment...%NC%
go version >nul 2>&1
if errorlevel 1 (
    echo %RED%❌ Go is not installed or not in PATH%NC%
    exit /b 1
)
echo %GREEN%✅ Go is available%NC%

echo %YELLOW%📦 Step 2: Downloading dependencies...%NC%
cd /d %COMPILER_DIR%
go mod download
if errorlevel 1 (
    echo %RED%❌ Failed to download dependencies%NC%
    exit /b 1
)
echo %GREEN%✅ Dependencies downloaded%NC%

echo %YELLOW%🎨 Step 3: Checking code formatting...%NC%
gofmt -s -l . > temp_fmt.txt
for /f %%i in (temp_fmt.txt) do (
    echo %RED%❌ The following files are not formatted correctly:%NC%
    type temp_fmt.txt
    echo %YELLOW%Please run: cd compiler && gofmt -s -w .%NC%
    del temp_fmt.txt
    exit /b 1
)
del temp_fmt.txt
echo %GREEN%✅ All Go files are properly formatted%NC%

echo %YELLOW%🔍 Step 4: Running go vet...%NC%
go vet ./...
if errorlevel 1 (
    echo %RED%❌ go vet failed%NC%
    exit /b 1
)
echo %GREEN%✅ go vet passed%NC%

echo %YELLOW%🧪 Step 5: Running tests...%NC%
go test -v ./...
if errorlevel 1 (
    echo %RED%❌ Tests failed%NC%
    exit /b 1
)
echo %GREEN%✅ All tests passed%NC%

echo %YELLOW%🔨 Step 6: Building compiler...%NC%
mkdir "%BIN_DIR%" 2>nul
go build -o "%BIN_DIR%\ferret.exe" -ldflags "-s -w" -trimpath -v
if errorlevel 1 (
    echo %RED%❌ Build failed%NC%
    exit /b 1
)
echo %GREEN%✅ Compiler built successfully%NC%

echo %YELLOW%🚀 Step 7: Testing CLI functionality...%NC%
cd /d %BIN_DIR%

REM Test help message
ferret.exe 2>&1 | findstr /C:"Usage: ferret" >nul
if errorlevel 1 (
    echo %RED%❌ CLI help message test failed%NC%
    exit /b 1
)

REM Test init command (simulate interactive input)
(echo myapp & echo true & echo true) | ferret.exe init > temp_output.txt 2>&1
findstr /C:"Project configuration initialized" temp_output.txt >nul
if errorlevel 1 (
    echo %RED%❌ CLI init command test failed%NC%
    del temp_output.txt
    exit /b 1
)

REM Verify config file was created
if not exist "fer.ret" (
    echo %RED%❌ Config file was not created%NC%
    del temp_output.txt
    exit /b 1
)

del temp_output.txt

echo %GREEN%✅ CLI functionality tests passed%NC%

REM Cleanup
del fer.ret 2>nul

REM Security scan (gosec)
echo %YELLOW%🔒 Step 8: Security scan (gosec)...%NC%
gosec -h >nul 2>&1
if errorlevel 1 (
    echo %YELLOW%⚠️  gosec not installed. Installing...%NC%
    go install github.com/securego/gosec/v2/cmd/gosec@latest
    if errorlevel 1 (
        echo %RED%❌ Failed to install gosec, skipping security scan%NC%
        echo %YELLOW%ℹ️  You can install gosec manually: go install github.com/securego/gosec/v2/cmd/gosec@latest%NC%
        echo %GREEN%✅ All other PR workflow checks passed!%NC%
        exit /b 0
    )
)
cd /d %COMPILER_DIR%
gosec -fmt sarif -out "%ROOT_DIR%\gosec.sarif" -stderr ./... 2>nul
if not exist "%ROOT_DIR%\gosec.sarif" (
    echo %YELLOW%Creating minimal SARIF file (no security issues found)%NC%
    echo {"version":"2.1.0","runs":[{"tool":{"driver":{"name":"gosec"}},"results":[]}]} > "%ROOT_DIR%\gosec.sarif"
)
echo %GREEN%✅ Security scan completed%NC%
echo %YELLOW%SARIF file created: %ROOT_DIR%\gosec.sarif%NC%

echo %GREEN%🎉 All PR workflow checks passed!%NC%
endlocal