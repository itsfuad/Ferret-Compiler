#!/bin/bash

# Change to compiler directory (we're in scripts, so go up one level then into compiler)
cd ../compiler

# Clear the screen
clear
echo "Formatting code..."

# Format the code
go fmt ./...

if [ $? -eq 0 ]; then
    echo "✅ Formatting successful"
else
    echo "❌ Formatting failed"
    exit 1
fi
