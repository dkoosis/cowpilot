#!/bin/bash

echo "🔧 Testing linter fixes..."
cd /Users/vcto/Projects/cowpilot

echo "📋 Running go vet..."
if go vet ./...; then
    echo "✅ go vet passed"
    echo ""
    echo "📋 Running linter..."
    if make lint > /dev/null 2>&1; then
        echo "✅ Linter passed - all issues resolved"
        echo ""
        echo "🧪 Testing OAuth spec test build..."
        cd cmd/oauth_spec_test
        if go build -o oauth-spec-test main.go; then
            echo "✅ OAuth test builds successfully"
            echo ""
            echo "🎯 All linting issues fixed - ready for OAuth spec test!"
            echo ""
            echo "📋 Final steps:"
            echo "  1. chmod +x run-test.sh" 
            echo "  2. ./run-test.sh"
            echo "  3. Register http://localhost:8090/mcp in Claude.ai"
            echo "  4. Watch logs to determine OAuth spec compliance"
        else
            echo "❌ OAuth test build failed"
            exit 1
        fi
    else
        echo "❌ Linter still failing"
        make lint
        exit 1
    fi
else
    echo "❌ go vet still failing"
    go vet ./...
    exit 1
fi
