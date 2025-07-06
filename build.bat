@echo off
cd compiler/cmd

go build -o ../bin/ferret.exe -ldflags "-s -w" -trimpath -v