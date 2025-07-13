@echo off
cls
echo Testing Ferret Codegen...
echo.

cd ..\compiler

echo Compiling simple.fer to assembly...
go run cmd/main.go compile ../app/simple.fer --output ../app/simple.s

if %errorlevel% equ 0 (
    echo.
    echo ✓ Compilation successful!
    echo Generated assembly file: ../app/simple.s
    echo.
    echo Assembly output:
    echo ----------------------------------------
    type ..\app\simple.s
    echo ----------------------------------------
) else (
    echo.
    echo ✗ Compilation failed!
    exit /b 1
)

echo.
echo Test completed.