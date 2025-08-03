#!/bin/bash
# Quick build and format check

set -e

cd /Users/vcto/Projects/cowpilot

echo "Step 1: Running gofmt..."
gofmt -w .

echo "Step 2: Running go mod tidy..."
go mod tidy

echo "Step 3: Building RTM server..."
go build -o bin/rtm-server ./cmd/rtm

echo "Step 4: Running tests..."
go test ./internal/longrunning/...

echo ""
echo "✓ Build successful! All checks passed."
echo ""
echo "The RTM server with long-running task support has been built."
echo "Binary location: bin/rtm-server"
