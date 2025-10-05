@echo off
REM Example usage of Symbol Query Server

echo === Symbol Query Server Example ===
echo.

REM Check if symquery is built
if not exist "symquery.exe" (
    echo Building symquery...
    go build -o symquery.exe
    if errorlevel 1 (
        echo Build failed!
        exit /b 1
    )
)

set PROJECT=..\app

echo Starting Symbol Query Server for project: %PROJECT%
echo.
echo Available commands:
echo   help       - Show available commands
echo   stats      - Show compilation statistics
echo   query ^<sym^> - Find information about a symbol
echo   list       - List all symbols
echo   modules    - List all modules
echo   exit       - Exit the server
echo.
echo Starting interactive session...
echo.

REM Start the server
symquery.exe %PROJECT%
