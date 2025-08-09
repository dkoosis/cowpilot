#!/bin/bash

echo "Checking RTM deployment..."
echo ""

# First check if the app even exists
echo "=== Fly.io Apps ==="
fly apps list 2>/dev/null | grep -E "rtm|mcp" || echo "No RTM/MCP apps found on Fly.io"

echo ""
echo "=== Quick Actions ==="
echo ""
echo "If no app exists, deploy it:"
echo "  fly apps create rtm-mcp"
echo "  make deploy-rtm"
echo ""
echo "If app exists but is suspended/dead:"
echo "  fly scale count 1 -a rtm-mcp"
echo "  fly restart -a rtm-mcp"
echo ""
echo "To check detailed status:"
echo "  fly status -a rtm-mcp"
echo "  fly logs -a rtm-mcp"
