#!/bin/bash
cd /Users/vcto/Projects/cowpilot

echo "Building RTM server..."
go build -o bin/rtm-server ./cmd/rtm 2>&1

if [ $? -eq 0 ]; then
    echo "✓ Build successful"
else
    echo "✗ Build failed"
fi
