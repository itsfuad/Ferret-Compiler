#!/bin/bash

# Clear the screen
clear

cd ..

echo "Cleaning up imports..."
# Remove unused imports
go mod tidy

echo "Formatting code..."

# Format the code
go fmt ./compiler/...

if [ $? -eq 0 ]; then
    echo "✅ Formatting successful"
else
    echo "❌ Formatting failed"
    exit 1
fi
