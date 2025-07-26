#!/bin/bash
# MCP Inspector Integration Test - Verify compatibility with official MCP Inspector

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== RUN   MCP Inspector Integration Test${NC}"
echo -e "${BLUE}    --- Testing server compatibility with @modelcontextprotocol/inspector${NC}"

cd /Users/vcto/Projects/cowpilot

# Check if inspector is available
if ! npx @modelcontextprotocol/inspector --version &>/dev/null; then
    echo -e "${YELLOW}    ⚠️  @modelcontextprotocol/inspector not found${NC}"
    echo -e "${YELLOW}    --- SKIP  MCP Inspector Integration Test${NC}"
    exit 0
fi

# Build and start server
echo -e "${BLUE}    --- Building server...${NC}"
if go build -o ./bin/cowpilot ./cmd/cowpilot 2>/dev/null; then
    echo -e "${GREEN}        ✓ Build successful${NC}"
else
    echo -e "${RED}        ✗ Build failed${NC}"
    echo -e "${RED}--- FAIL  MCP Inspector Integration Test${NC}"
    exit 1
fi

# Start server
echo -e "${BLUE}    --- Starting server on localhost:8080...${NC}"
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

# Function to run inspector test
run_inspector_test() {
    local test_name="$1"
    shift
    local args=("$@")
    
    echo -e "${BLUE}    --- $test_name${NC}"
    
    # Run with timeout to prevent hanging
    if timeout 10s npx @modelcontextprotocol/inspector "${args[@]}" &>/dev/null; then
        echo -e "${GREEN}        ✓ Success${NC}"
        return 0
    else
        echo -e "${RED}        ✗ Failed or timed out${NC}"
        return 1
    fi
}

FAILED=0

# Test 1: Inspector version check
echo -e "${BLUE}    --- Inspector version check${NC}"
if npx @modelcontextprotocol/inspector --version &>/dev/null; then
    echo -e "${GREEN}        ✓ Inspector available${NC}"
else
    echo -e "${RED}        ✗ Inspector not available${NC}"
    ((FAILED++))
fi

# Test 2: Initialize with inspector (HTTP transport)
if ! run_inspector_test "Initialize protocol (HTTP transport)" \
    --cli http://localhost:8080/ \
    --transport http \
    --method initialize \
    --params '{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"inspector-test","version":"1.0"}}'; then
    ((FAILED++))
fi

# Test 3: List tools with inspector (HTTP transport)
if ! run_inspector_test "List tools (HTTP transport)" \
    --cli http://localhost:8080/ \
    --transport http \
    --method tools/list; then
    ((FAILED++))
fi

# Test 4: Call tool with inspector
if ! run_inspector_test "Call hello tool (HTTP transport)" \
    --cli http://localhost:8080/ \
    --transport http \
    --method tools/call \
    --tool-name hello; then
    ((FAILED++))
fi

# Test 5: List resources
if ! run_inspector_test "List resources (HTTP transport)" \
    --cli http://localhost:8080/ \
    --transport http \
    --method resources/list; then
    ((FAILED++))
fi

# Test 6: Test with /mcp endpoint (forces HTTP detection)
if ! run_inspector_test "List tools via /mcp endpoint" \
    --cli http://localhost:8080/mcp \
    --method tools/list; then
    ((FAILED++))
fi

# Test 7: SSE transport test (expected to fail/timeout with current configuration)
echo -e "${BLUE}    --- Testing SSE transport (5s timeout)${NC}"
if timeout 5s npx @modelcontextprotocol/inspector \
    --cli http://localhost:8080/ \
    --method tools/list &>/dev/null; then
    echo -e "${YELLOW}        ⚠️  SSE transport worked (unexpected)${NC}"
else
    echo -e "${GREEN}        ✓ SSE transport timed out as expected${NC}"
fi

# Cleanup
echo -e "${BLUE}    --- Stopping server...${NC}"
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

# Summary
echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}--- PASS  MCP Inspector Integration Test${NC}"
    exit 0
else
    echo -e "${RED}--- FAIL  MCP Inspector Integration Test ($FAILED tests failed)${NC}"
    exit 1
fi
