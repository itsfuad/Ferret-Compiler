@echo off
setlocal enabledelayedexpansion

REM Colors (for Windows, just plain text)
set RED=
set GREEN=
set YELLOW=
set NC=

echo %YELLOW% Running local CI workflow simulation...%NC%

REM Set up directories
set SCRIPT_DIR=%~dp0
set ROOT_DIR=%SCRIPT_DIR%..
set COMPILER_DIR=%ROOT_DIR%\compiler
set BIN_DIR=%ROOT_DIR%\bin

cd /d %ROOT_DIR%

echo %YELLOW% Step 1: Setting up environment...%NC%
go version >nul 2>&1
if errorlevel 1 (
    echo %RED% Go is not installed or not in PATH%NC%
    exit /b 1
)
echo %GREEN% Go is available%NC%

echo %YELLOW% Step 2: Downloading dependencies...%NC%
cd /d %ROOT_DIR%
go mod download
if errorlevel 1 (
    echo %RED% Failed to download dependencies%NC%
    exit /b 1
)
echo %GREEN% Dependencies downloaded%NC%

echo %YELLOW% Step 3: Checking code formatting...%NC%
cd /d %ROOT_DIR%
gofmt -s -l ./compiler > temp_fmt.txt
for /f %%i in (temp_fmt.txt) do (
    echo %RED% The following files are not formatted correctly:%NC%
    type temp_fmt.txt
    echo %YELLOW%Please run: gofmt -s -w ./compiler%NC%
    del temp_fmt.txt
    exit /b 1
)
del temp_fmt.txt
echo %GREEN% All Go files are properly formatted%NC%

echo %YELLOW% Step 4: Running go vet...%NC%
go vet ./compiler/...
if errorlevel 1 (
    echo %RED% go vet failed%NC%
    exit /b 1
)
echo %GREEN% go vet passed%NC%

echo %YELLOW% Step 5: Running tests...%NC%
go test -v ./compiler/...
if errorlevel 1 (
    echo %RED% Tests failed%NC%
    exit /b 1
)
echo %GREEN% All tests passed%NC%

echo %YELLOW% Step 6: Building compiler...%NC%
mkdir "%BIN_DIR%" 2>nul
go build -o "%BIN_DIR%\ferret.exe" -ldflags "-s -w" -trimpath -v ./compiler
if errorlevel 1 (
    echo %RED% Build failed%NC%
    exit /b 1
)
echo %GREEN% Compiler built successfully%NC%

echo %YELLOW% Step 7: Testing CLI functionality...%NC%
cd /d %BIN_DIR%

REM Test help message
ferret.exe 2>&1 | findstr /C:"Usage: ferret" >nul
if errorlevel 1 (
    echo %RED% CLI help message test failed%NC%
    exit /b 1
)

REM Test init command (simulate interactive input)
(echo myapp & echo true & echo true) | ferret.exe init > temp_output.txt 2>&1
findstr /C:"Project configuration initialized" temp_output.txt >nul
if errorlevel 1 (
    echo %RED% CLI init command test failed%NC%
    del temp_output.txt
    exit /b 1
)

REM Verify config file was created
if not exist "fer.ret" (
    echo %RED% Config file was not created%NC%
    del temp_output.txt
    exit /b 1
)

del temp_output.txt

echo %GREEN% CLI functionality tests passed%NC%

REM Cleanup
del fer.ret 2>nul

echo %GREEN% All CI workflow checks passed!%NC%
endlocal