@echo off

:: Clear the screen
cls

cd ..

echo Cleaning up imports...
:: Remove unused imports
go mod tidy

echo Formatting code...
:: Format the code
go fmt ./compiler/...

if errorlevel 1 (
    echo Formatting failed
) else (
    echo Formatting successful
)

