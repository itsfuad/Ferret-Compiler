@echo off

cd ../compiler
go build -o ..\bin\ferret.exe -ldflags "-s -w" -trimpath -v .