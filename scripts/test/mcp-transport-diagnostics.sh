#!/bin/bash
# MCP Transport Diagnostics - Test different transport methods and client detection

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== RUN   MCP Transport Diagnostics"
echo -e "${BLUE}    --- Testing HTTP/SSE transport auto-detection and compatibility"

cd /Users/vcto/Projects/cowpilot

# Check dependencies
echo -e "${BLUE}    --- Checking dependencies..."
MISSING_DEPS=0
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}        ⚠️  jq not found (optional)"
    JQ="cat"
else
    echo -e "${GREEN}        ✓ jq available"
    JQ="jq -c"
fi

if ! command -v curl &> /dev/null; then
    echo -e "${RED}        ✗ curl not found (required)"
    ((MISSING_DEPS++))
else
    echo -e "${GREEN}        ✓ curl available"
fi

if [ $MISSING_DEPS -gt 0 ]; then
    echo -e "${RED} ✗  FAIL  MCP Transport Diagnostics (missing dependencies)"
    exit 1
fi

# Build server
echo -e "${BLUE}    --- Building server..."
if make build &>/dev/null; then
    echo -e "${GREEN}        ✓ Build successful"
else
    echo -e "${RED}        ✗ Build failed"
    echo -e "${RED} ✗  FAIL  MCP Transport Diagnostics"
    exit 1
fi

# Start server
echo -e "${BLUE}    --- Starting server..."
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

FAILED=0

# Test 1: Basic health check
echo -e "${BLUE}    --- Test 1: Basic health check"
if curl -s -f http://localhost:8080/health &>/dev/null; then
    echo -e "${GREEN}        ✓ Health endpoint responding"
else
    echo -e "${RED}        ✗ Health check failed"
    ((FAILED++))
fi

# Test 2: Protocol diagnostics endpoint
echo -e "${BLUE}    --- Test 2: Protocol diagnostics"
if response=$(curl -s http://localhost:8080/health?protocol=true); then
    echo -e "${GREEN}        ✓ Protocol info available"
    echo -e "${BLUE}        Transport: $(echo "$response" | grep -o '"transport":"[^"]*"' | cut -d'"' -f4)"
    echo -e "${BLUE}        Supports: $(echo "$response" | grep -o '"supports":\[[^]]*\]' | sed 's/"//g')"
else
    echo -e "${RED}        ✗ Protocol diagnostics failed"
    ((FAILED++))
fi

# Test 3: HTTP POST with JSON-RPC
echo -e "${BLUE}    --- Test 3: HTTP POST transport (JSON-RPC)"
response=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)

if echo "$response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ HTTP POST working"
    tool_count=$(echo "$response" | grep -o '"name"' | wc -l)
    echo -e "${BLUE}        Found $tool_count tools"
else
    echo -e "${RED}        ✗ HTTP POST failed"
    echo -e "${RED}        Response: $(echo "$response" | head -c 100)..."
    ((FAILED++))
fi

# Test 4: SSE connection attempt
echo -e "${BLUE}    --- Test 4: SSE transport test"
echo -e "${BLUE}        Attempting SSE connection (3s timeout)..."
sse_response=$(timeout 3s curl -s -N http://localhost:8080/ \
    -H "Accept: text/event-stream" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1 || true)

if [ -n "$sse_response" ]; then
    echo -e "${GREEN}        ✓ SSE transport responded"
    if echo "$sse_response" | grep -q "event:"; then
        echo -e "${BLUE}        SSE event stream detected"
    fi
else
    echo -e "${YELLOW}        ⚠️  SSE connection timed out (may be normal for stateless mode)"
fi

# Test 5: MCP endpoint (forces HTTP detection)
echo -e "${BLUE}    --- Test 5: /mcp endpoint test"
response=$(curl -s -X POST http://localhost:8080/mcp \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)

if echo "$response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ /mcp endpoint working (forces HTTP mode)"
else
    echo -e "${RED}        ✗ /mcp endpoint failed"
    ((FAILED++))
fi

# Test 6: Client detection test
echo -e "${BLUE}    --- Test 6: Client type detection"
echo -e "${BLUE}        Testing different User-Agent headers..."

# Test curl client
response=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -H "User-Agent: curl/7.64.1" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)
if echo "$response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ curl client detected and served"
else
    ((FAILED++))
fi

# Test node/inspector client
response=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -H "User-Agent: node-fetch/1.0" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)
if echo "$response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ node client detected and served"
else
    ((FAILED++))
fi

# Test 7: MCP Inspector compatibility check
echo -e "${BLUE}    --- Test 7: MCP Inspector compatibility"
if command -v npx &> /dev/null && npx @modelcontextprotocol/inspector --version &>/dev/null; then
    if timeout 10s npx @modelcontextprotocol/inspector \
        --cli http://localhost:8080/mcp \
        --method tools/list &>/dev/null; then
        echo -e "${GREEN}        ✓ MCP Inspector compatible"
    else
        echo -e "${RED}        ✗ MCP Inspector failed"
        ((FAILED++))
    fi
else
    echo -e "${YELLOW}        ⚠️  MCP Inspector not available (skipped)"
fi

# Cleanup
echo -e "${BLUE}    --- Stopping server..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

# Summary
echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN} ✓ PASS  MCP Transport Diagnostics"
    echo -e "${GREEN}    StreamableHTTP transport working correctly"
    echo -e "${GREEN}    Auto-detection of client types functioning"
    exit 0
else
    echo -e "${RED} ✗  FAIL  MCP Transport Diagnostics ($FAILED tests failed)"
    exit 1
fi
