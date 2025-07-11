#!/bin/bash

# Local CI simulation script
# This script runs the same checks that the CI pipeline will run

set -e

echo "🚀 Running local CI simulation..."

# Change to compiler directory (we're in scripts, so go up one level then into compiler)
cd ../compiler

echo ""
echo "📦 Downloading dependencies..."
go mod download

echo ""
echo "🎨 Checking code formatting..."
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo "❌ The following files are not formatted correctly:"
    gofmt -s -l .
    echo ""
    echo "Please run 'gofmt -s -w .' to fix formatting issues."
    exit 1
else
    echo "✅ All Go files are properly formatted"
fi

echo ""
echo "🔍 Running go vet..."
go vet ./...
echo "✅ go vet passed"

echo ""
echo "🧪 Running tests..."
go test -v ./...
echo "✅ All tests passed"

echo ""
echo "🔨 Building compiler..."
go build -v ./cmd
echo "✅ Compiler built successfully"

echo ""
echo "🚀 Testing CLI functionality..."
go build -o ferret-test ./cmd

# Test help message
if ! ./ferret-test 2>&1 | grep -q "Usage: ferret"; then
    echo "❌ CLI help message test failed"
    exit 1
fi

# Test init command
mkdir -p test-project
if ! ./ferret-test init test-project 2>&1 | grep -q "Project configuration initialized"; then
    echo "❌ CLI init command test failed"
    exit 1
fi

# Verify config file was created
if [ ! -f "test-project/.ferret.json" ]; then
    echo "❌ Config file was not created"
    exit 1
fi

echo "✅ CLI functionality tests passed"

# Cleanup
rm -rf test-project ferret-test

echo ""
echo "🎉 All CI checks passed! Your code is ready for push."
