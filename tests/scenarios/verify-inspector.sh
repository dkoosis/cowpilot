#!/bin/bash
# Quick test to verify MCP inspector is working

echo "Testing MCP Inspector installation..."

if ! command -v npx &> /dev/null; then
    echo "❌ npx not found. Please install Node.js first."
    exit 1
fi

if npx @modelcontextprotocol/inspector --version &> /dev/null; then
    echo "✅ MCP Inspector is installed and working"
    echo ""
    echo "You can now run the E2E tests with:"
    echo "  make e2e-test-prod"
    echo ""
    echo "Or test manually with:"
    echo "  npx @modelcontextprotocol/inspector --cli https://mcp-adapters.fly.dev/ --method tools/list"
else
    echo "❌ MCP Inspector not found"
    echo ""
    echo "Please install it with:"
    echo "  npm install -g @modelcontextprotocol/inspector"
    exit 1
fi
