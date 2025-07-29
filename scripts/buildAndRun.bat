@echo off
cd ..\compiler

go build -o ..\bin\ferret.exe -ldflags "-s -w" -trimpath -v

echo Running project

cd ..\app
..\bin\ferret.exe cmd/start.fer --debug