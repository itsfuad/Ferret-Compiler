@echo off

:: Change to compiler directory (we're in scripts, so go up one level then into compiler)
cd ..\compiler

:: Clear the screen
cls

echo Cleaning up imports...
:: Remove unused imports
go mod tidy

echo Formatting code...
:: Format the code
go fmt ./...

if errorlevel 1 (
    echo Formatting failed
) else (
    echo Formatting successful
)

