#!/bin/bash
# Quick build script for RTM server

echo "Building RTM server..."
go build -o bin/rtm-server ./cmd/rtm

if [ $? -eq 0 ]; then
    echo "✓ Build successful: bin/rtm-server"
    echo ""
    echo "To test locally:"
    echo "  FLY_APP_NAME=local PORT=8081 ./bin/rtm-server"
else
    echo "✗ Build failed"
    exit 1
fi
