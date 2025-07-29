#!/bin/bash

echo "🔧 Testing both import and API fixes..."
cd /Users/vcto/Projects/cowpilot

echo "📋 Running go vet to check for errors..."
if go vet ./...; then
    echo "✅ go vet passed - all issues resolved"
    echo ""
    echo "📦 Testing Makefile..."
    if make help > /dev/null 2>&1; then
        echo "✅ Makefile working"
        echo ""
        echo "🧪 Testing OAuth spec test build..."
        cd cmd/oauth_spec_test
        if go build -o oauth-spec-test main.go; then
            echo "✅ OAuth test builds successfully"
            echo ""
            echo "🎯 All fixes verified - ready for OAuth spec test!"
            echo ""
            echo "📋 Final steps:"
            echo "  1. cd cmd/oauth_spec_test"
            echo "  2. chmod +x run-test.sh" 
            echo "  3. ./run-test.sh"
            echo "  4. Register http://localhost:8090/mcp in Claude.ai"
            echo "  5. Watch logs to determine OAuth spec compliance"
        else
            echo "❌ OAuth test build failed"
            exit 1
        fi
    else
        echo "❌ Makefile still has issues"
        make help
        exit 1
    fi
else
    echo "❌ go vet still failing"
    go vet ./...
    exit 1
fi
