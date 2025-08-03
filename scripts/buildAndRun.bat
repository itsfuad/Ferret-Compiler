@echo off

cd ..
go build -o bin\ferret.exe -ldflags "-s -w" -trimpath -v ./compiler

echo Running project

cd app
..\bin\ferret.exe cmd/start.fer --debug