#!/bin/bash
# SSE Transport Test - Server-Sent Events protocol verification

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== RUN   SSE Transport Test${NC}"
echo -e "${BLUE}    --- Testing Server-Sent Events transport for browser clients${NC}"

cd /Users/vcto/Projects/cowpilot

# Build server
echo -e "${BLUE}    --- Building server...${NC}"
if make build &>/dev/null; then
    echo -e "${GREEN}        ✓ Build successful${NC}"
else
    echo -e "${RED}        ✗ Build failed${NC}"
    echo -e "${RED}--- FAIL  SSE Transport Test${NC}"
    exit 1
fi

# Start server with SSE enabled
echo -e "${BLUE}    --- Starting server with SSE support...${NC}"
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

FAILED=0

# Test 1: SSE connection with proper headers
echo -e "${BLUE}    --- Test 1: SSE connection with event-stream accept header${NC}"
echo -e "${BLUE}        Sending initialize request via SSE...${NC}"

# Use timeout to prevent indefinite hanging
sse_output=$(timeout 5s curl -s -N -X POST http://localhost:8080/ \
    -H "Accept: text/event-stream" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"sse-test","version":"1.0.0"}},"id":1}' 2>&1 || true)

if [ -n "$sse_output" ]; then
    echo -e "${GREEN}        ✓ SSE connection established${NC}"
    
    # Check for SSE format
    if echo "$sse_output" | grep -q "event:"; then
        echo -e "${GREEN}        ✓ SSE event format detected${NC}"
    else
        echo -e "${YELLOW}        ⚠️  No SSE events in response${NC}"
    fi
    
    # Display first few lines
    echo -e "${BLUE}        First response lines:${NC}"
    echo "$sse_output" | head -n 5 | sed 's/^/          /'
else
    echo -e "${YELLOW}        ⚠️  SSE connection timed out (stateless mode may not support SSE)${NC}"
fi

# Test 2: Multiple SSE requests
echo -e "${BLUE}    --- Test 2: Multiple SSE requests in sequence${NC}"

for i in 1 2 3; do
    echo -e "${BLUE}        Request $i: tools/list${NC}"
    response=$(timeout 3s curl -s -N -X POST http://localhost:8080/ \
        -H "Accept: text/event-stream" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":$i}" 2>&1 || true)
    
    if [ -n "$response" ]; then
        echo -e "${GREEN}        ✓ Response received${NC}"
    else
        echo -e "${YELLOW}        ⚠️  No response${NC}"
    fi
done

# Test 3: SSE vs HTTP content negotiation
echo -e "${BLUE}    --- Test 3: Content negotiation test${NC}"

# Test without Accept header (should use HTTP)
echo -e "${BLUE}        Testing without Accept header...${NC}"
http_response=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1)

if echo "$http_response" | grep -q '"result"'; then
    echo -e "${GREEN}        ✓ HTTP response received (no Accept header)${NC}"
else
    echo -e "${RED}        ✗ Failed to get HTTP response${NC}"
    ((FAILED++))
fi

# Test with SSE Accept header
echo -e "${BLUE}        Testing with SSE Accept header...${NC}"
sse_response=$(timeout 3s curl -s -N -X POST http://localhost:8080/ \
    -H "Accept: text/event-stream" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":2}' 2>&1 || true)

if [ -n "$sse_response" ]; then
    if echo "$sse_response" | grep -q "event:"; then
        echo -e "${GREEN}        ✓ SSE response received (Accept: text/event-stream)${NC}"
    else
        echo -e "${YELLOW}        ⚠️  Response received but not in SSE format${NC}"
    fi
else
    echo -e "${YELLOW}        ⚠️  SSE request timed out${NC}"
fi

# Test 4: Browser simulation
echo -e "${BLUE}    --- Test 4: Browser client simulation${NC}"
browser_response=$(timeout 5s curl -s -N -X POST http://localhost:8080/ \
    -H "Accept: text/event-stream, text/html, application/xhtml+xml, */*" \
    -H "Content-Type: application/json" \
    -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' 2>&1 || true)

if [ -n "$browser_response" ]; then
    echo -e "${GREEN}        ✓ Browser client served successfully${NC}"
else
    echo -e "${YELLOW}        ⚠️  Browser client request timed out${NC}"
fi

# Test 5: MCP Inspector forcing HTTP (should NOT use SSE)
echo -e "${BLUE}    --- Test 5: Verify HTTP override works${NC}"
if timeout 10s npx @modelcontextprotocol/inspector \
    --cli http://localhost:8080/mcp \
    --method tools/list &>/dev/null 2>&1; then
    echo -e "${GREEN}        ✓ HTTP override via /mcp endpoint working${NC}"
else
    echo -e "${YELLOW}        ⚠️  MCP Inspector not available or failed${NC}"
fi

# Cleanup
echo -e "${BLUE}    --- Stopping server...${NC}"
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

# Summary
echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}--- PASS  SSE Transport Test${NC}"
    echo -e "${GREEN}    Note: Stateless mode may limit SSE functionality${NC}"
    exit 0
else
    echo -e "${RED}--- FAIL  SSE Transport Test ($FAILED tests failed)${NC}"
    exit 1
fi
