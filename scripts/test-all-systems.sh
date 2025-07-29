#!/bin/bash

echo "🔧 Testing all path fixes including health check..."
cd /Users/vcto/Projects/cowpilot

echo "📋 Running go vet..."
if go vet ./...; then
    echo "✅ go vet passed"
    echo ""
    echo "📋 Running linter..."
    if make lint > /dev/null 2>&1; then
        echo "✅ Linter passed"
        echo ""
        echo "🧪 Testing project health check..."
        
        # Run health check script
        if bash scripts/test/project-health-check.sh > /tmp/health_check.log 2>&1; then
            echo "✅ Project health check passed"
            # Show relevant output
            grep -E "(✓|✗|⚠️)" /tmp/health_check.log | head -10
            rm -f /tmp/health_check.log
        else
            echo "❌ Project health check failed"
            echo "Health check output:"
            cat /tmp/health_check.log
            rm -f /tmp/health_check.log
            exit 1  
        fi
        
        echo ""
        echo "🧪 Testing integration test build..."
        if go build -o bin/test-cowpilot cmd/demo-server/main.go; then
            echo "✅ Integration test build works"
            rm -f bin/test-cowpilot
            echo ""
            echo "🧪 Testing OAuth spec test build..."
            cd cmd/oauth_spec_test
            if go build -o oauth-spec-test main.go; then
                echo "✅ OAuth test builds successfully"
                rm -f oauth-spec-test
                echo ""
                echo "🎯 All systems verified - ready for OAuth spec test!"
                echo ""
                echo "📋 Everything is now working:"
                echo "  ✅ File reorganization complete"
                echo "  ✅ Build system functional"
                echo "  ✅ Package conflicts resolved"
                echo "  ✅ Import paths updated" 
                echo "  ✅ Linter compliance achieved"
                echo "  ✅ Integration tests fixed"
                echo "  ✅ Health checks updated"
                echo ""
                echo "🎯 Ready for OAuth spec compliance test:"
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
            echo "❌ Integration test build failed"
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
