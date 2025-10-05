#!/bin/bash
# Example usage of Symbol Query Server

echo "=== Symbol Query Server Example ==="
echo ""

# Check if symquery is built
if [ ! -f "symquery.exe" ] && [ ! -f "symquery" ]; then
    echo "Building symquery..."
    go build -o symquery
    if [ $? -ne 0 ]; then
        echo "Build failed!"
        exit 1
    fi
fi

SYMQUERY="./symquery"
if [ -f "symquery.exe" ]; then
    SYMQUERY="./symquery.exe"
fi

PROJECT="../app"

echo "Starting Symbol Query Server for project: $PROJECT"
echo ""
echo "Available commands:"
echo "  help       - Show available commands"
echo "  stats      - Show compilation statistics"
echo "  query <sym> - Find information about a symbol"
echo "  list       - List all symbols"
echo "  modules    - List all modules"
echo "  exit       - Exit the server"
echo ""
echo "Starting interactive session..."
echo ""

# Start the server
$SYMQUERY "$PROJECT"
