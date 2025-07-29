#!/bin/bash

echo "🔧 Testing integration test path fix..."
cd /Users/vcto/Projects/cowpilot

echo "📋 Running go vet..."
if go vet ./...; then
    echo "✅ go vet passed"
    echo ""
    echo "📋 Running linter..."
    if make lint > /dev/null 2>&1; then
        echo "✅ Linter passed"
        echo ""
        echo "🧪 Testing integration test script fix..."
        
        # Test the integration test build command in isolation
        echo "  Testing build command: go build -o bin/test-cowpilot cmd/demo-server/main.go"
        if go build -o bin/test-cowpilot cmd/demo-server/main.go; then
            echo "✅ Integration test build command works"
            rm -f bin/test-cowpilot
            echo ""
            echo "🧪 Testing OAuth spec test build..."
            cd cmd/oauth_spec_test
            if go build -o oauth-spec-test main.go; then
                echo "✅ OAuth test builds successfully"
                rm -f oauth-spec-test
                echo ""
                echo "🎯 All fixes verified - everything ready!"
                echo ""
                echo "📋 You can now run:"
                echo "  • Integration tests: make integration-test-local"
                echo "  • OAuth spec test: cd cmd/oauth_spec_test && ./run-test.sh"
                echo ""
                echo "🔍 For OAuth spec compliance test:"
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
            echo "❌ Integration test build command failed"
            echo "This suggests the demo-server path is still incorrect"
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
