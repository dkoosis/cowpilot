#!/bin/bash
cd /Users/vcto/Projects/cowpilot

echo "Building RTM server..."
if go build -o build/rtm-server ./cmd/rtm; then
    echo "✓ Build successful!"
    echo ""
    echo "Ready to deploy: make deploy-rtm"
else
    echo "✗ Build failed"
    exit 1
fi
