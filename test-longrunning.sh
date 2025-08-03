#!/bin/bash
# Quick test of the longrunning package

cd /Users/vcto/Projects/cowpilot

echo "Running tests for longrunning package..."
go test -v ./internal/longrunning/...

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ All tests passed"
else
    echo ""
    echo "❌ Tests failed"
    exit 1
fi
