@echo off
REM Local CI simulation script for Windows
REM This script runs the same checks that the CI pipeline will run

echo 🚀 Running local CI simulation...

REM Change to compiler directory (we're in scripts, so go up one level then into compiler)
cd ..\compiler

echo.
echo 📦 Downloading dependencies...
go mod download

echo.
echo 🎨 Checking code formatting...
for /f "delims=" %%i in ('gofmt -s -l .') do (
    echo ❌ The following files are not formatted correctly:
    gofmt -s -l .
    echo.
    echo Please run 'gofmt -s -w .' to fix formatting issues.
    exit /b 1
)
echo ✅ All Go files are properly formatted

echo.
echo 🔍 Running go vet...
go vet ./...
if errorlevel 1 (
    echo ❌ go vet failed
    exit /b 1
)
echo ✅ go vet passed

echo.
echo 🧪 Running tests...
go test -v ./...
if errorlevel 1 (
    echo ❌ Tests failed
    exit /b 1
)
echo ✅ All tests passed

echo.
echo 🔨 Building compiler...
go build -v ./cmd
if errorlevel 1 (
    echo ❌ Build failed
    exit /b 1
)
echo ✅ Compiler built successfully

echo.
echo 🚀 Testing CLI functionality...
go build -o ferret-test.exe ./cmd

REM Test help message
ferret-test.exe > temp_output.txt 2>&1
findstr /C:"Usage: ferret" temp_output.txt >nul
if errorlevel 1 (
    echo ❌ CLI help message test failed
    del temp_output.txt
    del ferret-test.exe
    exit /b 1
)

REM Test init command
mkdir test-project 2>nul
ferret-test.exe init test-project > temp_output.txt 2>&1
findstr /C:"Project configuration initialized" temp_output.txt >nul
if errorlevel 1 (
    echo ❌ CLI init command test failed
    del temp_output.txt
    rmdir /s /q test-project 2>nul
    del ferret-test.exe
    exit /b 1
)

REM Verify config file was created
if not exist "test-project\.ferret.json" (
    echo ❌ Config file was not created
    del temp_output.txt
    rmdir /s /q test-project 2>nul
    del ferret-test.exe
    exit /b 1
)

echo ✅ CLI functionality tests passed

REM Cleanup
del temp_output.txt
rmdir /s /q test-project 2>nul
del ferret-test.exe

echo.
echo 🎉 All CI checks passed! Your code is ready for push.
