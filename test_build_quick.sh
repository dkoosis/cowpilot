#!/bin/bash
# Quick build test
cd /Users/vcto/Projects/cowpilot
go build -o bin/cowpilot cmd/cowpilot/main.go
echo "Build completed with exit code: $?"
