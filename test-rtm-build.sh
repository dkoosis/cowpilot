#!/bin/bash
cd /Users/vcto/Projects/cowpilot

echo "Building RTM server with fixes..."
if go build -o build/rtm-server ./cmd/rtm; then
    echo "✓ Build successful!"
    echo ""
    echo "Enhanced RTM tools ready:"
    echo "• 19 total tools (8 original + 11 atomic)"
    echo "• Async batch operations"
    echo "• Smart search with caching"
    echo "• Position-based operations"
    echo ""
    echo "Next: make deploy-rtm"
else
    echo "✗ Build failed:"
    go build -o build/rtm-server ./cmd/rtm 2>&1
fi
