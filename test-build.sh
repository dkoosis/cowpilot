#!/bin/bash
cd /Users/vcto/Projects/cowpilot
echo "Testing build..."
go build -o bin/test-rtm cmd/rtm/main.go 2>&1
echo "Exit code: $?"
