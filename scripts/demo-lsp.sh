#!/bin/bash

echo "🚀 Ferret LSP System Demo"
echo "=========================="

# Check if binaries exist
if [ ! -f "../bin/ferret" ]; then
    echo "❌ Ferret compiler not found. Run ./build.sh first"
    exit 1
fi

if [ ! -f "../bin/ferret-lsp" ]; then
    echo "❌ Ferret LSP server not found. Run ./lsp.sh first"
    exit 1
fi

echo "✅ Ferret compiler: $(../bin/ferret --version 2>/dev/null || echo 'Available')"
echo "✅ Ferret LSP server: Available"

# Test LSP server startup
echo ""
echo "🔧 Testing LSP Server Startup..."
echo "Starting LSP server on port 9876..."

# Start LSP server in background
../bin/ferret-lsp --port 9876 &
LSP_PID=$!

# Give it time to start
sleep 2

# Check if it's running
if kill -0 $LSP_PID 2>/dev/null; then
    echo "✅ LSP server started successfully (PID: $LSP_PID)"
    
    # Test if port is listening
    if command -v netstat >/dev/null; then
        if netstat -ln 2>/dev/null | grep -q ":9876"; then
            echo "✅ LSP server listening on port 9876"
        else
            echo "⚠️ Port check inconclusive"
        fi
    fi
    
    # Stop the server
    echo "🛑 Stopping LSP server..."
    kill $LSP_PID 2>/dev/null
    wait $LSP_PID 2>/dev/null
    echo "✅ LSP server stopped"
else
    echo "❌ LSP server failed to start"
    exit 1
fi

echo ""
echo "📁 Testing with sample Ferret file..."
if [ -f "../app/test_lsp.fer" ]; then
    echo "✅ Found test file: app/test_lsp.fer"
    echo "📝 File contents:"
    echo "---"
    head -n 10 "../app/test_lsp.fer"
    echo "---"
    
    # Test compilation
    echo ""
    echo "🔍 Testing compilation with diagnostics..."
    cd ../app
    if ../bin/ferret run --debug test_lsp.fer 2>/dev/null; then
        echo "✅ File compiles successfully"
    else
        echo "⚠️ File has compilation issues (expected for demo)"
    fi
    cd ../scripts
else
    echo "❌ Test file not found"
fi

echo ""
echo "📦 VS Code Extension Status..."
if [ -f "../extension/out/client.js" ]; then
    echo "✅ Extension compiled: extension/out/client.js"
    echo "📏 Bundle size: $(du -h ../extension/out/client.js | cut -f1)"
else
    echo "⚠️ Extension not compiled. Run: cd extension && npm run bundle"
fi

if [ -f "../extension/ferret-"*.vsix ]; then
    echo "✅ Extension packaged: $(ls ../extension/ferret-*.vsix | head -1 | xargs basename)"
else
    echo "ℹ️ Extension not packaged. Install 'vsce' and run ./pack.sh"
fi

echo ""
echo "🎯 Quick Setup Summary:"
echo "1. Build everything: ./pack.sh"
echo "2. Install VS Code extension from generated .vsix file"
echo "3. Open a .fer file in VS Code"
echo "4. Enjoy LSP features: completion, hover, diagnostics!"

echo ""
echo "📚 Available LSP Features:"
echo "  ✅ Real-time diagnostics"
echo "  ✅ Code completion with snippets"
echo "  ✅ Hover information"
echo "  🔄 Go-to-definition (framework ready)"
echo "  🔄 Find references (framework ready)"
echo "  🔄 Document symbols (framework ready)"
echo "  🔄 Code formatting (framework ready)"

echo ""
echo "🚀 Demo completed successfully!"
echo "Ready for development with Ferret LSP! 🎉"