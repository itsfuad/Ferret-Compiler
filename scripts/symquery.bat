@echo off
REM Build and run the Symbol Query Server

echo 🔨 Building symquery...
cd "%~dp0..\symquery"
go build -o symquery.exe

if %ERRORLEVEL% NEQ 0 (
    echo ❌ Build failed
    exit /b 1
)

echo ✅ Build successful!
echo.

if "%~1"=="" (
    echo Usage: symquery.bat ^<project-root^> [--json] [--debug]
    echo.
    echo Examples:
    echo   symquery.bat ..\app              # Interactive mode
    echo   symquery.bat ..\app --debug      # With debug output
    echo   symquery.bat ..\app --json       # JSON mode for programmatic access
    exit /b 1
)

echo 🚀 Starting Symbol Query Server...
symquery.exe %*
