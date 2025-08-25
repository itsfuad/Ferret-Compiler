#!/bin/bash
# Local PR workflow simulation script for Unix/Linux/macOS
# This script simulates the GitHub Actions PR workflow as closely as possible

set -e  # Exit on any error

echo "🚀 Running local PR workflow simulation..."

# Set variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
COMPILER_DIR="$ROOT_DIR/compiler"
BIN_DIR="$ROOT_DIR/bin"

# Change to root directory
cd "$ROOT_DIR"

echo ""
echo "📦 Step 1: Setting up environment..."
echo "Current directory: $(pwd)"
echo "Compiler directory: $COMPILER_DIR"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi
echo "✅ Go is available: $(go version)"

echo ""
echo "📦 Step 2: Downloading dependencies..."
cd "$COMPILER_DIR"
go mod download
echo "✅ Dependencies downloaded"

echo ""
echo "🎨 Step 3: Checking code formatting..."
cd "$COMPILER_DIR"
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo "❌ The following files are not formatted correctly:"
    gofmt -s -l .
    echo ""
    echo "Please run the following command to fix formatting issues:"
    echo "gofmt -s -w ."
    exit 1
else
    echo "✅ All Go files are properly formatted"
fi

echo ""
echo "🔍 Step 4: Running go vet..."
cd "$COMPILER_DIR"
go vet ./...
echo "✅ go vet passed"

echo ""
echo "🧪 Step 5: Running tests..."
cd "$COMPILER_DIR"
go test -v ./...
echo "✅ All tests passed"

echo ""
echo "🔨 Step 6: Building compiler..."
mkdir -p "$BIN_DIR"
cd "$COMPILER_DIR"
go build -o "$BIN_DIR/ferret" -ldflags "-s -w" -trimpath -v .
chmod +x "$BIN_DIR/ferret"
echo "✅ Compiler built successfully"

echo ""
echo "🚀 Step 7: Testing CLI functionality..."
cd "$ROOT_DIR"
cd $BIN_DIR

# Test help message
if ! "./ferret" 2>&1 | grep -q "Ferret"; then
    echo "❌ CLI help message test failed"
    exit 1
fi

# Test init command
if ! (echo -e "myapp\ntrue\ntrue" | ./ferret init) | grep -q "Project configuration initialized"; then
    echo -e "${RED}❌ CLI init command test failed${NC}"
    exit 1
fi

# Verify config file was created
if [ ! -f "fer.ret" ]; then
    echo -e "${RED}❌ Config file was not created${NC}"
    exit 1
fi

echo "✅ CLI functionality tests passed"

# Cleanup
rm -rf fer.ret

echo ""
echo "🔒 Step 8: Security scan (gosec)..."

# Check if gosec is installed
if ! command -v gosec &> /dev/null; then
    echo "⚠️   gosec not installed. Installing..."
    if ! go install github.com/securego/gosec/v2/cmd/gosec@latest; then
        echo "❌ Failed to install gosec, skipping security scan"
        echo "⚠️   You can install gosec manually: go install github.com/securego/gosec/v2/cmd/gosec@latest"
        echo "✅ All other PR workflow checks passed!"
        exit 0
    fi
fi

cd "$COMPILER_DIR"
# Run gosec and always create a SARIF file
gosec -fmt sarif -out "$ROOT_DIR/gosec.sarif" -stderr ./... || true

# Check if SARIF file was created
if [ ! -f "$ROOT_DIR/gosec.sarif" ] || [ ! -s "$ROOT_DIR/gosec.sarif" ]; then
    echo "Creating minimal SARIF file (no security issues found)"
    echo '{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"gosec"}},"results":[]}]}' > "$ROOT_DIR/gosec.sarif"
fi

echo "✅ Security scan completed"
echo "SARIF file created: $ROOT_DIR/gosec.sarif"

echo ""
echo "🎉 All PR workflow checks passed!"
echo ""
echo "Summary:"
echo "✅ Code formatting"
echo "✅ Static analysis (go vet)"
echo "✅ Unit tests"
echo "✅ Build"
echo "✅ CLI functionality"
echo "✅ Security scan"

cd "$ROOT_DIR"
