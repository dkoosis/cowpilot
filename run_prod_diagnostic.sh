#!/bin/bash

# Quick run of production diagnostics
cd /Users/vcto/Projects/cowpilot

echo "Running OAuth Production Diagnostics"
echo "====================================="
echo ""

# Run the OAuth trace tool
echo "1. Testing OAuth endpoints..."
go run scripts/diagnostics/oauth_trace.go

echo ""
echo "2. Checking if flyctl is available..."
if command -v flyctl &> /dev/null; then
    echo "✓ flyctl found"
    echo ""
    echo "3. Fetching recent production logs..."
    flyctl logs --app rtm --tail | head -100 | grep -E "\[OAuth|ERROR|401|token|callback" || echo "No OAuth-related logs found"
else
    echo "✗ flyctl not installed"
    echo "Install with: brew install flyctl"
fi

echo ""
echo "Diagnostic complete. Look for any ✗ marks above."
