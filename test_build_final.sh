#!/bin/bash
# Build test with detailed output
set -e

echo "=== Cowpilot Build Test ==="
echo "Current directory: $(pwd)"
cd /Users/vcto/Projects/cowpilot

echo -e "\n1. Cleaning..."
make clean

echo -e "\n2. Formatting code..."
gofmt -w .

echo -e "\n3. Running go vet..."
go vet ./...

echo -e "\n4. Building..."
go build -o bin/cowpilot cmd/cowpilot/main.go

echo -e "\n=== BUILD SUCCESSFUL ==="
echo "Binary created at: bin/cowpilot"
echo "Size: $(ls -lh bin/cowpilot | awk '{print $5}')"
echo -e "\nYou can now run:"
echo "  ./bin/cowpilot              # For stdio mode"
echo "  npx @modelcontextprotocol/inspector ./bin/cowpilot  # For testing"
