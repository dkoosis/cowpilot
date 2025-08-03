#!/bin/bash
cd /Users/vcto/Projects/cowpilot

echo "Testing RTM enhanced compilation..."
if go test -c ./internal/rtm -o /dev/null 2>&1; then
    echo "✓ RTM package compiles successfully"
else
    echo "✗ Compilation errors:"
    go test -c ./internal/rtm 2>&1
fi

echo ""
echo "Building RTM server..."
if go build -o build/rtm-server ./cmd/rtm 2>&1; then
    echo "✓ RTM server builds successfully"
    echo ""
    echo "Enhanced RTM features ready:"
    echo "• Batch operations with job queue"
    echo "• Smart search with position tracking" 
    echo "• Intelligent task creation"
    echo "• 19 total tools (8 original + 11 atomic)"
else
    echo "✗ Build errors:"
    go build -o build/rtm-server ./cmd/rtm 2>&1
fi
