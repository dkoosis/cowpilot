#!/bin/bash
# Quick test script for RTM enhancements

echo "Building RTM server with enhanced tools..."
cd /Users/vcto/Projects/cowpilot

# Build the RTM server
if go build -o build/rtm-server ./cmd/rtm; then
    echo "✓ Build successful"
else
    echo "✗ Build failed"
    exit 1
fi

echo "RTM server built successfully with:"
echo "- 8 original tools"
echo "- 11 new atomic tools"
echo "- Job queue for batch operations"
echo "- Smart search with caching"
echo "- Intelligent task creation"

echo ""
echo "To test:"
echo "1. Deploy: make deploy-rtm"
echo "2. Register in Claude: https://rtm.fly.dev/mcp"
echo "3. Try: 'Search for my priority tasks due this week'"
echo "4. Then: 'Set tasks 1,3,5 due Wednesday'"
