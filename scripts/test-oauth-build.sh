#!/bin/bash

echo "🔧 Making OAuth test script executable..."
chmod +x /Users/vcto/Projects/cowpilot/cmd/oauth_spec_test/run-test.sh

echo "🧪 Testing build..."
cd /Users/vcto/Projects/cowpilot/cmd/oauth_spec_test

echo "📦 Building OAuth spec test server..."
go build -o oauth-spec-test main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo ""
    echo "🎯 Ready to test Claude.ai OAuth spec compliance"
    echo "📁 Location: cmd/oauth_spec_test/"
    echo "🚀 Run: ./run-test.sh"
    echo ""
    echo "📋 What this will test:"
    echo "  ✅ OAuth 2.0 Protected Resource Metadata (RFC 9728)"
    echo "  ✅ Resource Indicators (RFC 8707)"
    echo "  ✅ Separate authorization server pattern"
    echo "  ✅ Resource server only MCP pattern"
    echo ""
    echo "🎯 Next steps:"
    echo "  1. Run: ./run-test.sh"
    echo "  2. Try to register http://localhost:8090/mcp in Claude.ai"
    echo "  3. Observe Claude's OAuth behavior in the logs"
    echo ""
else
    echo "❌ Build failed. Check errors above."
    exit 1
fi
