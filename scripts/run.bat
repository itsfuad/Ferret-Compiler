@echo off
cls

echo Building Ferret...

cd ../compiler
if not exist ../bin (
    mkdir ../bin
)
go build -o ..\bin\ferret.exe -ldflags "-s -w" -trimpath -v .

if %errorlevel% neq 0 (
    echo Build failed. Exiting...
    exit /b %errorlevel%
)
echo Running Ferret...
cd ../bin
ferret.exe run -debug
if %errorlevel% neq 0 (
    echo Execution failed. Exiting...
    exit /b %errorlevel%
)
echo Ferret executed successfully.
