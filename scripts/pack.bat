echo off
cls
echo Building Language Server Protocol (LSP) for Ferret...
cd ../lsp
go build -o ..\bin\ferret-lsp.exe -ldflags "-s -w" -trimpath -v .
cd ../extension
echo Building Ferret extension...
vsce package