@echo off
echo Testing Ferret Compiler - Debug and Import Features
echo =====================================================

echo.
echo 1. Testing Debug Output for Last Expression:
echo ---------------------------------------------
cd /d "%~dp0\..\compiler"
go run .\cmd\main.go ..\app\simple.fer --debug

echo.
echo 2. Testing Import Functionality with Debug:
echo -------------------------------------------
go run .\cmd\main.go ..\app\test_import.fer --debug

echo.
echo 3. Testing Custom Output Path:
echo ------------------------------
go run .\cmd\main.go ..\app\simple.fer --debug -o ..\app\demo_output.asm

echo.
echo 4. Verifying Custom Output File:
echo --------------------------------
type ..\app\demo_output.asm

echo.
echo Demo completed successfully!
