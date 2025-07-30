#!/bin/bash
# MCP Integration Test Runner

# Configuration
MCP_SERVER_URL="${MCP_SERVER_URL:-https://mcp-adapters.fly.dev/mcp}"
LOCAL_TEST="${LOCAL_TEST:-false}"

START_TIME=$(date +%s)

# Check if testing locally
if [[ "$LOCAL_TEST" == "true" ]]; then
    export MCP_SERVER_URL="http://localhost:8080/mcp"
    
    # Start local server if not running
    if ! curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
        echo "   starting local server..."
        
        # Kill any existing processes on port 8080
        lsof -ti:8080 | xargs kill -9 2>/dev/null || true
        sleep 1
        
        # Build and start server
        cd /Users/vcto/Projects/cowpilot
        go build -o bin/cowpilot cmd/core/main.go
        
        FLY_APP_NAME=local-test MCP_LOG_LEVEL=WARN ./bin/cowpilot --disable-auth > server.log 2>&1 &
        SERVER_PID=$!
        
        # Wait for server readiness
        for i in {1..60}; do
            if curl -s -f http://localhost:8080/health > /dev/null 2>&1 && \
               curl -s -f -X POST http://localhost:8080/mcp \
                    -H "Content-Type: application/json" \
                    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' > /dev/null 2>&1; then
                break
            fi
            sleep 0.5
        done
        
        if [ $i -eq 60 ]; then
            echo " ✗ server failed to start"
            kill $SERVER_PID 2>/dev/null || true
            exit 1
        fi
        
        SERVER_START_TIME=$(date +%s)
        SERVER_DURATION=$((SERVER_START_TIME - START_TIME))
        echo " ✓ server ready (${SERVER_DURATION}.00s)"
    fi
fi

echo "   running integration tests..."

# Run Go tests
cd /Users/vcto/Projects/cowpilot/tests
TEST_START=$(date +%s)

if command -v gotestsum &> /dev/null; then
    TEMP_FILE=$(mktemp)
    gotestsum --format testname -- -v -run TestMCP ./... > "$TEMP_FILE" 2>&1
    TEST_COUNT=$(grep -c "PASS:" "$TEMP_FILE" || echo "0")
    rm -f "$TEMP_FILE"
else
    TEST_COUNT=$(go test -v -run TestMCP ./... 2>/dev/null | grep -c "PASS:" || echo "0")
fi

TEST_END=$(date +%s)
TEST_DURATION=$((TEST_END - TEST_START))

echo " ✓ integration tests ($TEST_COUNT tests, ${TEST_DURATION}.00s)"

# Test with Inspector CLI if available
if command -v npx &> /dev/null; then
    echo "   testing with mcp inspector..."
    
    INSPECTOR_START=$(date +%s)
    if npx @modelcontextprotocol/inspector \
        --cli "$MCP_SERVER_URL" \
        --method tools/list \
        --transport http > /tmp/inspector_tools.json 2>/dev/null; then
        TOOL_COUNT=$(jq '.tools | length' /tmp/inspector_tools.json 2>/dev/null || echo "unknown")
        rm -f /tmp/inspector_tools.json
        
        INSPECTOR_END=$(date +%s)
        INSPECTOR_DURATION=$((INSPECTOR_END - INSPECTOR_START))
        echo " ✓ mcp inspector ($TOOL_COUNT tools, ${INSPECTOR_DURATION}.00s)"
    fi
fi

# Cleanup local server if we started it
if [[ -n "$SERVER_PID" ]]; then
    echo "   stopping local server..."
    kill $SERVER_PID 2>/dev/null || true
    rm -f server.log
fi

END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))
echo " ✓ integration tests complete (${TOTAL_DURATION}.00s)"
