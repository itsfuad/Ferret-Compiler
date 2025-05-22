@echo off

:: Check if the current directory is not "compiler", then change to it
for %%I in ("%CD%") do if /I not "%%~nxI"=="compiler" (
    cd compiler
)

:: Clear the screen
cls
echo Formatting code...
:: Format the code
go fmt ./...

if errorlevel 1 (
    echo Formatting failed
) else (
    echo Formatting successful
)

