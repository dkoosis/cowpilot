#!/bin/bash

# Script to demonstrate the enhanced test output

echo "🧪 Running tests with human-readable output..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if gotestsum is available
if command -v gotestsum &> /dev/null; then
    echo "Using gotestsum for enhanced output..."
    gotestsum --format testname -- -v ./internal/mcp/... ./cmd/cowpilot/... | grep -E "(✓|✗|SCENARIO:|GIVEN:|WHEN:|THEN:|📋 TESTED:|===)" || true
else
    echo "Using standard go test with filtering..."
    go test -v ./internal/mcp/... ./cmd/cowpilot/... | grep -E "(✓|✗|SCENARIO:|GIVEN:|WHEN:|THEN:|📋 TESTED:|===)" || true
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Test output complete"
