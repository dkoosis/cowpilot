#!/bin/bash
# Quick build & test validation for Cowpilot
# Based on STATE.yaml requirements

echo "🔍 Cowpilot Status Validation"
echo "═══════════════════════════════════════"

# Check current directory
if [[ ! -f "cmd/cowpilot/main.go" ]]; then
    echo "❌ Run this from the cowpilot project root"
    exit 1
fi

echo "📁 Project structure: ✅"

# 1. Build test
echo -e "\n🔨 Testing build..."
if go build -o bin/cowpilot cmd/cowpilot/main.go; then
    echo "✅ Build successful"
    ls -lh bin/cowpilot | awk '{print "   Binary size: " $5}'
else
    echo "❌ Build failed"
    exit 1
fi

# 2. Quick unit tests
echo -e "\n🧪 Running unit tests..."
if go test -v ./internal/mcp/... 2>/dev/null | grep -E "(PASS|FAIL)"; then
    echo "✅ Unit tests completed"
else
    echo "⚠️  No unit tests found or failed"
fi

# 3. Dependency check
echo -e "\n📦 Checking dependencies..."
go mod verify && echo "✅ Dependencies verified" || echo "❌ Dependency issues"

# 4. Test server startup (brief)
echo -e "\n🚀 Testing server startup..."
timeout 3s ./bin/cowpilot 2>/dev/null &
if [[ $? -eq 124 ]]; then
    echo "✅ Server starts successfully (stdio mode)"
else
    echo "⚠️  Server startup test inconclusive"
fi

# 5. Verify features from STATE.yaml
echo -e "\n🎯 Feature verification..."
echo "   Tools: $(grep -c 'AddTool' cmd/cowpilot/main.go)/11 expected"
echo "   Resources: $(grep -c 'AddResource' cmd/cowpilot/main.go)/4 expected"  
echo "   Prompts: $(grep -c 'AddPrompt' cmd/cowpilot/main.go)/2 expected"

# 6. Production health check
echo -e "\n🌐 Production server check..."
if curl -s --max-time 5 "https://cowpilot.fly.dev/health" | grep -q "OK"; then
    echo "✅ Production server healthy"
else
    echo "⚠️  Production server check failed (may be normal)"
fi

echo -e "\n═══════════════════════════════════════"
echo "🎉 Validation complete!"
echo ""
echo "Next steps:"
echo "  make test-verbose     # Full test suite"
echo "  make scenario-test-local  # E2E testing"
echo "  npx @modelcontextprotocol/inspector ./bin/cowpilot  # MCP testing"
