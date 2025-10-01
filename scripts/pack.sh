#!/bin/bash

echo "Building Ferret Compiler, LSP Server and VS Code Extension..."

# Build the main compiler
echo "Building Ferret compiler..."
cd ../compiler
go build -o ../bin/ferret -ldflags "-s -w" -trimpath -v .
if [ $? -ne 0 ]; then
    echo "✗ Failed to build compiler"
    exit 1
fi
echo "✓ Compiler built successfully"

# Build the LSP server
echo "Building LSP server..."
cd ../lsp
go build -o ../bin/ferret-lsp -ldflags "-s -w" -trimpath -v .
if [ $? -ne 0 ]; then
    echo "✗ Failed to build LSP server"
    exit 1
fi
echo "✓ LSP server built successfully"

# Build the VS Code extension
echo "Building VS Code extension..."
cd ../extension

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    echo "Installing npm dependencies..."
    npm install
    if [ $? -ne 0 ]; then
        echo "✗ Failed to install npm dependencies"
        exit 1
    fi
fi

# Compile TypeScript
echo "Compiling TypeScript..."
npm run compile
if [ $? -ne 0 ]; then
    echo "✗ Failed to compile TypeScript"
    exit 1
fi

# Bundle the extension
echo "Bundling extension..."
npm run bundle
if [ $? -ne 0 ]; then
    echo "✗ Failed to bundle extension"
    exit 1
fi

echo "✓ All components built successfully!"
echo "  - Compiler: bin/ferret"
echo "  - LSP Server: bin/ferret-lsp"
echo "  - VS Code Extension: extension/out/client.js"

# Optional: Package the extension
if command -v vsce &> /dev/null; then
    echo "Packaging VS Code extension..."
    npx vsce package
    if [ $? -eq 0 ]; then
        echo "✓ Extension packaged successfully"
    else
        echo "⚠ Extension packaging failed, but build was successful"
    fi
else
    echo "ℹ Install 'vsce' to package the extension: npm install -g vsce"
fi