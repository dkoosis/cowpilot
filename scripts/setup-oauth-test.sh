#!/bin/bash

echo "Setting executable permissions..."
chmod +x /Users/vcto/Projects/cowpilot/cmd/oauth_spec_test/run-test.sh

echo ""
echo "🧪 MCP OAuth Spec Compliance Test Ready!"
echo "========================================"
echo ""
echo "📁 Location: cmd/oauth_spec_test/"
echo "🚀 Run test: cd cmd/oauth_spec_test && ./run-test.sh"
echo ""
echo "📋 What this tests:"
echo "✅ OAuth 2.0 Protected Resource Metadata (RFC 9728)"
echo "✅ Resource Indicators (RFC 8707)" 
echo "✅ Separate auth server pattern"
echo "✅ Resource server only MCP pattern"
echo ""
echo "🎯 Purpose: Determine if Claude.ai supports new MCP OAuth spec"
echo "💡 Result: Guides whether to fix current auth or migrate to new spec"
echo ""
echo "Ready to test Claude.ai OAuth spec compliance!"
