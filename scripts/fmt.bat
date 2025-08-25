@echo off

:: Clear the screen
cls

cd ../compiler

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

