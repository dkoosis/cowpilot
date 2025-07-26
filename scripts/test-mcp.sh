#!/bin/bash

cd /Users/vcto/Projects/cowpilot

echo "🧪 Testing internal/mcp package directly..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Run just the MCP tests
go test -v ./internal/mcp/... 2>&1

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Exit code: $?"
