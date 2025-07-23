@echo off
cd ..\compiler\cmd

go build -o ../../bin/ferret.exe -ldflags "-s -w" -trimpath -v

echo Running project

cd ..\..\app
ferret.exe cmd/start.fer --debug