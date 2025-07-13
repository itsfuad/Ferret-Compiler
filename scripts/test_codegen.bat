@echo off
cls
echo Testing Ferret Codegen...
echo.

cd ..\compiler

echo Ensuring config file exists...
if not exist .ferret.json (
    copy ..\app\.ferret.json .
)

echo Compiling simple.fer to assembly...
set SIMPLE_PATH=%cd%\..\app\simple.fer
go run cmd/main.go "%SIMPLE_PATH%" --debug

if %errorlevel% equ 0 (
    echo.
    echo ✓ Compilation successful!
    echo Generated assembly file: ../app/output.asm
    echo.
    echo Assembly output:
    echo ----------------------------------------
    type ..\app\output.asm
    echo ----------------------------------------
) else (
    echo.
    echo ✗ Compilation failed!
    exit /b 1
)

echo.
echo Test completed.