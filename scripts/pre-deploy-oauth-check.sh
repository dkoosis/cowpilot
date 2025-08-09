#!/bin/bash
# Pre-deployment OAuth validation

set -e

echo "🔍 Running OAuth registration tests..."

# Run the actual tests we just created
go test -v ./tests/integration -run TestClaudeOAuthCompliance -tags=integration

if [ $? -ne 0 ]; then
    echo "❌ OAuth tests FAILED - deployment blocked"
    echo "The OAuth endpoints are broken and Claude.ai registration will fail"
    exit 1
fi

echo "✅ OAuth tests passed - safe to deploy"
