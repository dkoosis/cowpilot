#!/bin/bash

echo "🔧 Testing Makefile fixes..."
cd /Users/vcto/Projects/cowpilot

echo "📋 Testing make help..."
if make help > /dev/null 2>&1; then
    echo "✅ Makefile working - help command successful"
else 
    echo "❌ Makefile still has issues"
    make help
    exit 1
fi

echo ""
echo "📦 Testing OAuth spec test build..."
cd cmd/oauth_spec_test

echo "Building OAuth test server..."
if go build -o oauth-spec-test main.go; then
    echo "✅ OAuth test server builds successfully"
    echo ""
    echo "🎯 Ready to test Claude.ai OAuth spec compliance!"
    echo ""
    echo "📋 Next steps:"
    echo "  1. chmod +x run-test.sh"
    echo "  2. ./run-test.sh"
    echo "  3. Register http://localhost:8090/mcp in Claude.ai"
    echo "  4. Watch logs to see which OAuth pattern Claude uses"
    echo ""
else
    echo "❌ OAuth test build failed"
    exit 1
fi
