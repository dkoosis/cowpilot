#!/bin/bash
# Quick setup for E2E testing

echo "Setting up E2E tests for mcp adapters..."
echo ""

# Make scripts executable from project root
cd "$(dirname "$0")/../.." || exit 1

echo "1. Making test scripts executable..."
chmod +x tests/e2e/*.sh
echo "   ✓ Done"
echo ""

echo "2. Checking for Node.js..."
if command -v node &> /dev/null; then
    echo "   ✓ Node.js found: $(node --version)"
else
    echo "   ❌ Node.js not found. Please install Node.js first."
    exit 1
fi
echo ""

echo "3. Checking for MCP Inspector..."
if npx @modelcontextprotocol/inspector --version &> /dev/null 2>&1; then
    echo "   ✓ MCP Inspector is available"
else
    echo "   ⚠️  MCP Inspector not found globally"
    echo "   Installing now..."
    npm install -g @modelcontextprotocol/inspector
    if [ $? -eq 0 ]; then
        echo "   ✓ MCP Inspector installed successfully"
    else
        echo "   ❌ Failed to install MCP Inspector"
        echo "   Try running: sudo npm install -g @modelcontextprotocol/inspector"
        exit 1
    fi
fi
echo ""

echo "4. Ready to run tests!"
echo "   - Test production:  make e2e-test-prod"
echo "   - Test local:       make e2e-test-local"
echo "   - Custom server:    MCP_SERVER_URL=https://your-server/ make e2e-test"
echo ""
echo "   Or run directly:"
echo "   ./tests/e2e/mcp_scenarios.sh https://mcp-adapters.fly.dev/"
