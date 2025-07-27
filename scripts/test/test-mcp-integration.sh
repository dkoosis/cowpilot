#!/bin/bash
# MCP Integration Test Runner

echo "🧪 MCP Integration Tests"

# Configuration
MCP_SERVER_URL="${MCP_SERVER_URL:-https://cowpilot.fly.dev/mcp}"
LOCAL_TEST="${LOCAL_TEST:-false}"

# Check if testing locally
if [[ "$LOCAL_TEST" == "true" ]]; then
    echo "📍 Testing locally at http://localhost:8080/mcp"
    export MCP_SERVER_URL="http://localhost:8080/mcp"
    
    # Start local server if not running
    if ! curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
        echo "🚀 Starting local server..."
        go run cmd/cowpilot/main.go --disable-auth &
        SERVER_PID=$!
        
        # Wait for server
        for i in {1..30}; do
            if curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
                echo "✅ Server ready"
                break
            fi
            sleep 0.5
        done
    fi
else
    echo "📍 Testing deployed server at $MCP_SERVER_URL"
fi

echo ""
echo "🔧 Running tests..."

# Run Go tests
cd tests
go test -v ./... -run TestMCP

# Test with Inspector CLI if available
if command -v npx &> /dev/null; then
    echo ""
    echo "🔍 Testing with MCP Inspector CLI..."
    
    # Test tools/list
    npx @modelcontextprotocol/inspector \
        --cli "$MCP_SERVER_URL" \
        --method tools/list \
        --transport http
    
    # Test hello tool
    echo ""
    echo "📞 Calling hello tool..."
    npx @modelcontextprotocol/inspector \
        --cli "$MCP_SERVER_URL" \
        --method tools/call \
        --params '{"name":"hello","arguments":{}}' \
        --transport http
fi

# Cleanup local server if we started it
if [[ -n "$SERVER_PID" ]]; then
    echo ""
    echo "🛑 Stopping local server..."
    kill $SERVER_PID 2>/dev/null
fi

echo ""
echo "✅ Tests complete!"
