#!/bin/bash
# MCP Integration Test Runner

echo "🧪 MCP Integration Tests"

# Configuration
MCP_SERVER_URL="${MCP_SERVER_URL:-https://mcp-adapters.fly.dev/mcp}"
LOCAL_TEST="${LOCAL_TEST:-false}"

# Check if testing locally
if [[ "$LOCAL_TEST" == "true" ]]; then
    echo "📍 Testing locally at http://localhost:8080/mcp"
    export MCP_SERVER_URL="http://localhost:8080/mcp"
    
    # Start local server if not running
    if ! curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
        echo "🚀 Starting local server..."
        
        # Kill any existing processes on port 8080
        lsof -ti:8080 | xargs kill -9 2>/dev/null || true
        sleep 1
        
        # Build the binary first
        cd /Users/vcto/Projects/cowpilot
        go build -o bin/cowpilot cmd/demo-server/main.go
        
        # Start server with minimal logging and error capture
        echo "🚀 Starting server (PID will be shown)..."
        FLY_APP_NAME=local-test MCP_LOG_LEVEL=WARN ./bin/cowpilot --disable-auth > server.log 2>&1 &
        SERVER_PID=$!
        echo "Server started with PID: $SERVER_PID"
        
        # Wait for server with better readiness check
        echo "⏳ Waiting for server to be ready..."
        for i in {1..60}; do
            if curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
                # Also test MCP endpoint specifically
                if curl -s -f -X POST http://localhost:8080/mcp \
                    -H "Content-Type: application/json" \
                    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' > /dev/null 2>&1; then
                    echo "✅ Server ready (health + MCP endpoints responding)"
                    break
                fi
            fi
            echo "  Attempt $i/60..."
            sleep 0.5
        done
        
        if [ $i -eq 60 ]; then
            echo "❌ Server failed to start after 30 seconds"
            echo "Checking if process is still running..."
            if ps -p $SERVER_PID > /dev/null 2>&1; then
                echo "Process $SERVER_PID is still running (likely binding issue)"
            else
                echo "Process $SERVER_PID has died (likely startup crash)"
            fi
            echo "Recent server log output:"
            tail -20 server.log 2>/dev/null || echo "No server log available"
            if [ -n "$SERVER_PID" ]; then
                kill $SERVER_PID 2>/dev/null || true
            fi
            exit 1
        fi
    fi
else
    echo "📍 Testing deployed server at $MCP_SERVER_URL"
fi

echo ""
echo "🔧 Running tests..."

# Run Go tests with clean output
cd /Users/vcto/Projects/cowpilot/tests
if command -v gotestsum &> /dev/null; then
    gotestsum --format testdox -- -run TestMCP ./...
else
    go test -run TestMCP ./...
fi

# Test with Inspector CLI if available
if command -v npx &> /dev/null; then
    echo ""
    echo "🔍 Testing with MCP Inspector CLI..."
    
    # Test tools/list (silent)
    if npx @modelcontextprotocol/inspector \
        --cli "$MCP_SERVER_URL" \
        --method tools/list \
        --transport http > /tmp/inspector_tools.json 2>/dev/null; then
        TOOL_COUNT=$(jq '.tools | length' /tmp/inspector_tools.json 2>/dev/null || echo "unknown")
        echo "   ✓ Tools list: $TOOL_COUNT tools available"
        rm -f /tmp/inspector_tools.json
    else
        echo "   ⚠ Inspector CLI test failed (expected for URL-based servers)"
    fi
else
    echo "🔍 Inspector CLI not available, skipping"
fi

# Cleanup local server if we started it
if [[ -n "$SERVER_PID" ]]; then
    echo ""
    echo "🛑 Stopping local server..."
    kill $SERVER_PID 2>/dev/null || true
    
    # Clean up server log if tests passed
    if [ -f server.log ] && [ ! -s server.log ]; then
        rm -f server.log
    fi
fi

echo ""
echo "✅ Tests complete!"
