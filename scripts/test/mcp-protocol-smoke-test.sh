#!/bin/bash
# MCP Protocol Smoke Test - Basic protocol verification via direct HTTP/JSON-RPC

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# gotestsum-style formatting
echo -e "${BLUE}=== RUN   MCP Protocol Smoke Test${NC}"
echo -e "${BLUE}    --- Testing basic MCP protocol operations via curl${NC}"

cd /Users/vcto/Projects/cowpilot

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}    ⚠️  jq not found, using raw output${NC}"
    JQ="cat"
else
    JQ="jq -c"
fi

# Build and start server
echo -e "${BLUE}    --- Building server...${NC}"
if go build -o ./bin/cowpilot ./cmd/cowpilot 2>/dev/null; then
    echo -e "${GREEN}        ✓ Build successful${NC}"
else
    echo -e "${RED}        ✗ Build failed${NC}"
    echo -e "${RED}--- FAIL  MCP Protocol Smoke Test${NC}"
    exit 1
fi

# Start server
echo -e "${BLUE}    --- Starting server...${NC}"
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

# Function to run test and format output
run_test() {
    local test_name="$1"
    local json_request="$2"
    
    echo -e "${BLUE}    --- $test_name${NC}"
    
    response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d "$json_request" \
        http://localhost:8080/ 2>/dev/null || echo '{"error": "curl failed"}')
    
    if echo "$response" | grep -q '"error"'; then
        echo -e "${RED}        ✗ Failed: $(echo $response | $JQ | head -c 100)${NC}"
        return 1
    else
        echo -e "${GREEN}        ✓ Success${NC}"
        return 0
    fi
}

# Run tests
FAILED=0

# Test 1: Initialize
if ! run_test "Initialize protocol" \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke-test","version":"1.0"}}}'; then
    ((FAILED++))
fi

# Test 2: List tools
if ! run_test "List available tools" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'; then
    ((FAILED++))
fi

# Test 3: Call hello tool
if ! run_test "Call hello tool" \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"hello"}}'; then
    ((FAILED++))
fi

# Test 4: Call echo tool with arguments
if ! run_test "Call echo tool with arguments" \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{"message":"smoke test"}}}'; then
    ((FAILED++))
fi

# Test 5: List resources
if ! run_test "List available resources" \
    '{"jsonrpc":"2.0","id":5,"method":"resources/list"}'; then
    ((FAILED++))
fi

# Test 6: List prompts
if ! run_test "List available prompts" \
    '{"jsonrpc":"2.0","id":6,"method":"prompts/list"}'; then
    ((FAILED++))
fi

# Cleanup
echo -e "${BLUE}    --- Stopping server...${NC}"
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

# Summary
echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}--- PASS  MCP Protocol Smoke Test${NC}"
    exit 0
else
    echo -e "${RED}--- FAIL  MCP Protocol Smoke Test ($FAILED tests failed)${NC}"
    exit 1
fi
