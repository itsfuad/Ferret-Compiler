@echo off

cd ..
go build -o bin\ferret.exe -ldflags "-s -w" -trimpath -v ./compiler