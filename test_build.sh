#!/bin/bash
cd /Users/vcto/Projects/cowpilot
echo "=== Running go vet ==="
go vet ./...
VET_RESULT=$?

echo "=== Running go build ==="
go build -o bin/cowpilot cmd/cowpilot/main.go
BUILD_RESULT=$?

if [ $VET_RESULT -eq 0 ] && [ $BUILD_RESULT -eq 0 ]; then
    echo "✅ All checks passed!"
    exit 0
else
    echo "❌ Some checks failed."
    echo "Vet result: $VET_RESULT"
    echo "Build result: $BUILD_RESULT"
    exit 1
fi
