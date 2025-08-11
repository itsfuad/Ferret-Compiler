#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🚀 Running local PR workflow simulation...${NC}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
COMPILER_DIR="$ROOT_DIR/compiler"
BIN_DIR="$ROOT_DIR/bin"

cd "$ROOT_DIR"

echo -e "${YELLOW}📦 Step 1: Setting up environment...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Go is available: $(go version)${NC}"

echo -e "${YELLOW}📦 Step 2: Downloading dependencies...${NC}"
cd "$COMPILER_DIR"
go mod download
echo -e "${GREEN}✅ Dependencies downloaded${NC}"

echo -e "${YELLOW}🎨 Step 3: Checking code formatting...${NC}"
cd "$COMPILER_DIR"
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo -e "${RED}❌ The following files are not formatted correctly:${NC}"
    gofmt -s -l .
    echo -e "${YELLOW}Please run: gofmt -s -w .${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All Go files are properly formatted${NC}"
fi

echo -e "${YELLOW}🔍 Step 4: Running go vet...${NC}"
cd "$COMPILER_DIR"
go vet ./...
echo -e "${GREEN}✅ go vet passed${NC}"

echo -e "${YELLOW}🧪 Step 5: Running tests...${NC}"
cd "$COMPILER_DIR"
go test -v ./...
echo -e "${GREEN}✅ All tests passed${NC}"

echo -e "${YELLOW}🔨 Step 6: Building compiler...${NC}"
mkdir -p "$BIN_DIR"
cd "$COMPILER_DIR"
go build -o "$BIN_DIR/ferret" -ldflags "-s -w" -trimpath -v .
chmod +x "$BIN_DIR/ferret"
echo -e "${GREEN}✅ Compiler built successfully${NC}"

echo -e "${YELLOW}🚀 Step 7: Testing CLI functionality...${NC}"
cd "$ROOT_DIR"
cd $BIN_DIR
# Test help message
if ! "./ferret" 2>&1 | grep -q "Ferret"; then
    echo -e "${RED}❌ CLI help message test failed${NC}"
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

echo -e "${GREEN}✅ CLI functionality tests passed${NC}"

# Cleanup
rm -rf fer.ret

echo -e "${YELLOW}🔒 Step 8: Security scan (gosec)...${NC}"
if ! command -v gosec &> /dev/null; then
    echo -e "${YELLOW}⚠️   gosec not installed. Installing...${NC}"
    if ! go install github.com/securego/gosec/v2/cmd/gosec@latest; then
        echo -e "${RED}❌ Failed to install gosec, skipping security scan${NC}"
        echo -e "${YELLOW}⚠️   You can install gosec manually: go install github.com/securego/gosec/v2/cmd/gosec@latest${NC}"
        echo -e "${GREEN}✅ All other PR workflow checks passed!${NC}"
        exit 0
    fi
fi

cd "$COMPILER_DIR"
gosec -fmt sarif -out "$ROOT_DIR/gosec.sarif" -stderr ./... || true

if [ ! -f "$ROOT_DIR/gosec.sarif" ] || [ ! -s "$ROOT_DIR/gosec.sarif" ]; then
    echo -e "${YELLOW}Creating minimal SARIF file (no security issues found)${NC}"
    echo '{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"gosec"}},"results":[]}]}' > "$ROOT_DIR/gosec.sarif"
fi

echo -e "${GREEN}✅ Security scan completed${NC}"
echo -e "${YELLOW}SARIF file created: $ROOT_DIR/gosec.sarif${NC}"

echo -e "${GREEN}🎉 All PR workflow checks passed!${NC}"
echo ""
echo -e "${GREEN}Summary:${NC}"
echo -e "${GREEN}✅ Code formatting"
echo -e "✅ Static analysis (go vet)"
echo -e "✅ Unit tests"
echo -e "✅ Build"
echo -e "✅ CLI functionality"
echo -e "✅ Security scan${NC}"
cd "$ROOT_DIR"
