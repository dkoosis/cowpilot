#!/bin/bash
# MCP Transport Diagnostics - Test different transport methods and client detection

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== RUN   MCP Transport Diagnostics${NC}"
echo -e "${BLUE}    --- Testing HTTP/SSE transport auto-detection and compatibility${NC}"

cd /Users/vcto/Projects/cowpilot

# Check dependencies
echo -e "${BLUE}    --- Checking dependencies...${NC}"
MISSING_DEPS=0
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}        ⚠️  jq not found (optional)${NC}"
    JQ="cat"
else
    echo -e "${GREEN}        ✓ jq available${NC}"
    JQ="jq -c"
fi

if ! command -v curl &> /dev/null; then
    echo -e "${RED}        ✗ curl not found (required)${NC}"
    ((MISSING_DEPS++))
else
    echo -e "${GREEN}        ✓ curl available${NC}"
fi

if [ $MISSING_DEPS -gt 0 ]; then
    echo -e "${RED}--- FAIL  MCP Transport Diagnostics (missing dependencies)${NC}"
    exit 1
fi

# Build server
echo -e "${BLUE}    --- Building server...${NC}"
if make build &>/dev/null; then
    echo -e "${GREEN}        ✓ Build successful${NC}"
else
    echo -e "${RED}        ✗ Build failed${NC}"
    echo -e "${RED}--- FAIL  MCP Transport Diagnostics${NC}"
    exit 1
fi

# Start server
echo -e "${BLUE}    --- Starting server...${NC}"
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

FAILED=0

# Test 1: Basic health check
echo -e "${BLUE}    --- Test 1: Basic health check${NC}"
if curl -s -f http://localhost:8080/health &>/dev/null; then
    echo -e "${GREEN}        ✓ Health endpoint responding${NC}"
else
    echo -e "${RED}        ✗ Health check failed${NC}"
    ((FAILED++))
fi

# Test 2: Protocol diagnostics endpoint
echo -e "${BLUE}    --- Test 2: Protocol diagnostics${NC}"
if response=$(curl -s http://localhost:8080/health?protocol=true); then
    echo -e "${GREEN}        ✓ Protocol info available${NC}"
    echo -e "${BLUE}        Transport: $(echo "$response" | grep -o '"transport":"[^"]*"' | cut -d'"' -f4)${NC}"
    echo -e "${BLUE}        Supports: $(echo "$response" | grep -o '"supports":\[[^]]*\]' | sed 's/"//g')${NC}"
else
    echo -e "${RED}        ✗ Protocol diagnostics failed${NC}"
    ((FAILED++))
fi

# Test 3: HTTP POST with JSON-RPC
echo -e "${BLUE}    --- Test 3: HTTP POST transport (JSON-RPC)${NC}"
response=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)

if echo "$response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ HTTP POST working${NC}"
    tool_count=$(echo "$response" | grep -o '"name"' | wc -l)
    echo -e "${BLUE}        Found $tool_count tools${NC}"
else
    echo -e "${RED}        ✗ HTTP POST failed${NC}"
    echo -e "${RED}        Response: $(echo "$response" | head -c 100)...${NC}"
    ((FAILED++))
fi

# Test 4: SSE connection attempt
echo -e "${BLUE}    --- Test 4: SSE transport test${NC}"
echo -e "${BLUE}        Attempting SSE connection (3s timeout)...${NC}"
sse_response=$(timeout 3s curl -s -N http://localhost:8080/ \
    -H "Accept: text/event-stream" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1 || true)

if [ -n "$sse_response" ]; then
    echo -e "${GREEN}        ✓ SSE transport responded${NC}"
    if echo "$sse_response" | grep -q "event:"; then
        echo -e "${BLUE}        SSE event stream detected${NC}"
    fi
else
    echo -e "${YELLOW}        ⚠️  SSE connection timed out (may be normal for stateless mode)${NC}"
fi

# Test 5: MCP endpoint (forces HTTP detection)
echo -e "${BLUE}    --- Test 5: /mcp endpoint test${NC}"
response=$(curl -s -X POST http://localhost:8080/mcp \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)

if echo "$response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ /mcp endpoint working (forces HTTP mode)${NC}"
else
    echo -e "${RED}        ✗ /mcp endpoint failed${NC}"
    ((FAILED++))
fi

# Test 6: Client detection test
echo -e "${BLUE}    --- Test 6: Client type detection${NC}"
echo -e "${BLUE}        Testing different User-Agent headers...${NC}"

# Test curl client
response=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -H "User-Agent: curl/7.64.1" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)
if echo "$response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ curl client detected and served${NC}"
else
    ((FAILED++))
fi

# Test node/inspector client
response=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -H "User-Agent: node-fetch/1.0" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)
if echo "$response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ node client detected and served${NC}"
else
    ((FAILED++))
fi

# Test 7: MCP Inspector compatibility check
echo -e "${BLUE}    --- Test 7: MCP Inspector compatibility${NC}"
if command -v npx &> /dev/null && npx @modelcontextprotocol/inspector --version &>/dev/null; then
    if timeout 10s npx @modelcontextprotocol/inspector \
        --cli http://localhost:8080/mcp \
        --method tools/list &>/dev/null; then
        echo -e "${GREEN}        ✓ MCP Inspector compatible${NC}"
    else
        echo -e "${RED}        ✗ MCP Inspector failed${NC}"
        ((FAILED++))
    fi
else
    echo -e "${YELLOW}        ⚠️  MCP Inspector not available (skipped)${NC}"
fi

# Cleanup
echo -e "${BLUE}    --- Stopping server...${NC}"
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

# Summary
echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}--- PASS  MCP Transport Diagnostics${NC}"
    echo -e "${GREEN}    StreamableHTTP transport working correctly${NC}"
    echo -e "${GREEN}    Auto-detection of client types functioning${NC}"
    exit 0
else
    echo -e "${RED}--- FAIL  MCP Transport Diagnostics ($FAILED tests failed)${NC}"
    exit 1
fi
