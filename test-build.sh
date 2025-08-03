#!/bin/bash
# Quick build test for RTM server with long-running tasks

set -e

echo "Running quick build test..."

# Install dependencies
echo "Installing dependencies..."
go mod download

# Run unit tests for longrunning package
echo "Testing longrunning package..."
go test ./internal/longrunning/...

# Build RTM server
echo "Building RTM server..."
go build -o bin/rtm-server ./cmd/rtm

echo "Build successful!"
echo ""
echo "To run the server locally:"
echo "  RTM_API_KEY=your_key RTM_API_SECRET=your_secret ./bin/rtm-server"
echo ""
echo "Or with fly.io:"
echo "  make deploy-rtm"
