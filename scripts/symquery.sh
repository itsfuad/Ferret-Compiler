#!/bin/bash
# Build and run the Symbol Query Server

set -e

echo "🔨 Building symquery..."
cd "$(dirname "$0")/../symquery"
go build -o symquery

echo "✅ Build successful!"
echo ""

if [ "$1" == "" ]; then
    echo "Usage: ./symquery.sh <project-root> [--json] [--debug]"
    echo ""
    echo "Examples:"
    echo "  ./symquery.sh ../app              # Interactive mode"
    echo "  ./symquery.sh ../app --debug      # With debug output"
    echo "  ./symquery.sh ../app --json       # JSON mode for programmatic access"
    exit 1
fi

echo "🚀 Starting Symbol Query Server..."
./symquery "$@"
