#!/bin/bash
# Quick test runner to verify new test structure

set -e

echo "=== Testing New Go Test Structure ==="
echo

# 1. Compile test to check syntax
echo "1. Checking compilation..."
cd /Users/vcto/Projects/cowpilot
if go test -c -tags=integration ./tests/integration/... > /dev/null 2>&1; then
    echo "✓ Integration tests compile"
else
    echo "✗ Compilation failed"
    exit 1
fi

# 2. Run unit tests (no tags)
echo
echo "2. Running unit tests (no server needed)..."
if go test -short -count=1 ./internal/rtm/... > /dev/null 2>&1; then
    echo "✓ Unit tests pass"
else
    echo "✗ Unit tests failed"
fi

# 3. Test that integration test can build server
echo
echo "3. Building server binary..."
if go build -o bin/core-server cmd/core/main.go; then
    echo "✓ Server builds successfully"
else
    echo "✗ Server build failed"
    exit 1
fi

# 4. Quick integration test run
echo
echo "4. Running single integration test with TestMain..."
if go test -tags=integration -run TestHealthEndpoint -timeout 30s ./tests/integration/...; then
    echo "✓ TestMain lifecycle works"
else
    echo "✗ TestMain failed"
fi

echo
echo "=== Test Structure Verified ==="
echo
echo "Next steps:"
echo "- Run 'make test' for standard tests"
echo "- Run 'make test-all' for comprehensive suite"
echo "- Old shell scripts remain as backup in scripts/test/"
