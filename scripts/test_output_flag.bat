@echo off
cls
echo Testing Ferret Codegen with -o flag...
echo.

cd ..\compiler

echo Ensuring config file exists...
if not exist .ferret.json (
    copy ..\app\.ferret.json .
)

echo Compiling simple.fer to custom output path...
set SIMPLE_PATH=%cd%\..\app\simple.fer
set OUTPUT_PATH=%cd%\..\app\custom_output.s
go run cmd/main.go "%SIMPLE_PATH%" -o "%OUTPUT_PATH%" --debug

if %errorlevel% equ 0 (
    echo.
    echo ✓ Compilation successful!
    echo Generated assembly file: %OUTPUT_PATH%
    echo.
    echo Assembly output:
    echo ----------------------------------------
    type ..\app\custom_output.s
    echo ----------------------------------------
) else (
    echo.
    echo ✗ Compilation failed!
    exit /b 1
)

echo.
echo Test completed.
