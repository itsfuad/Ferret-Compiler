#!/bin/bash

echo "Building Ferret LSP Server..."

# Change to LSP directory
cd ../lsp

# Build the LSP server
go build -o ../bin/ferret-lsp -ldflags "-s -w" -trimpath -v .

if [ $? -eq 0 ]; then
    echo "✓ LSP server built successfully: bin/ferret-lsp"
else
    echo "✗ Failed to build LSP server"
    exit 1
fi