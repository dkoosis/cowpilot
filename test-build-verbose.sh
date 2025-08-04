#!/bin/bash
cd /Users/vcto/Projects/cowpilot
echo "Downloading dependencies..."
go mod download
echo "Tidying modules..."
go mod tidy
echo "Building RTM server..."
go build -v -o bin/test-rtm cmd/rtm/main.go 2>&1
echo "Exit code: $?"
