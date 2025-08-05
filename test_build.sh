#!/bin/bash

echo "Testing RTM server build with pagination changes..."
cd /Users/vcto/Projects/cowpilot

# Build RTM server
go build -o /tmp/rtm-test cmd/rtm/main.go
if [ $? -eq 0 ]; then
    echo "✓ RTM server builds successfully"
    rm /tmp/rtm-test
else
    echo "✗ Build failed"
    exit 1
fi

echo "✓ All builds successful - pagination implementation compiles correctly"
