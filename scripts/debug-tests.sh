#!/bin/bash

echo "🧪 Running internal/mcp tests..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# First, let's see raw output
echo "Raw test output:"
go test -v ./internal/mcp/... 2>&1 | head -20

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Filtered output:"
go test -v ./internal/mcp/... 2>&1 | grep -E "(✓|✗|SCENARIO:|GIVEN:|WHEN:|THEN:|📋 TESTED:|===|RUN|PASS|FAIL)" || echo "No matching patterns found"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Checking if tests exist:"
find ./internal/mcp -name "*_test.go" -exec echo "Found: {}" \;
