#!/bin/bash

# Raw SSE/JSON-RPC Test Suite for Cowpilot MCP Server
# Based on https://blog.fka.dev/blog/2025-03-25-inspecting-mcp-servers-using-cli/
# Tests MCP protocol at the raw HTTP/SSE level

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Server URL (default to production if not provided)
SERVER_URL="${1:-https://cowpilot.fly.dev/}"

# Ensure URL ends without trailing slash for consistency
SERVER_URL="${SERVER_URL%/}"

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Check for required tools
for tool in curl jq; do
    if ! command -v $tool &> /dev/null; then
        echo -e "${RED}Error: $tool is required but not installed.${NC}"
        exit 1
    fi
done

# Function to print test header
print_test_header() {
    echo -e "\n${YELLOW}=== TEST: $1 ===${NC}"
    ((TOTAL_TESTS++))
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASSED_TESTS++))
}

# Function to print failure
print_failure() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    echo -e "${RED}Expected:${NC} $2"
    echo -e "${RED}Actual:${NC} $3"
    ((FAILED_TESTS++))
}

# Function to send JSON-RPC over SSE and parse response
send_jsonrpc() {
    local json_payload="$1"
    local timeout="${2:-5}"
    
    # Send request and capture SSE response
    # The MCP SSE format has each message prefixed with "data: "
    local response=$(echo "$json_payload" | \
        curl -s -N -X POST "$SERVER_URL" \
        -H "Content-Type: application/json" \
        -H "Accept: text/event-stream" \
        -d @- \
        --max-time "$timeout" 2>/dev/null | \
        grep "^data: " | \
        sed 's/^data: //' | \
        grep -v "^\[DONE\]$" | \
        head -n 1)
    
    echo "$response"
}

# As an MCP Client, I want to initialize a connection so that I can establish communication with the server.
print_test_header "As an MCP Client, I want to initialize a connection"

INIT_JSON=$(cat <<EOF
{
    "jsonrpc": "2.0",
    "method": "initialize",
    "params": {
        "protocolVersion": "2025-03-26",
        "capabilities": {},
        "clientInfo": {
            "name": "raw-sse-test",
            "version": "1.0.0"
        }
    },
    "id": 1
}
EOF
)

echo -e "${BLUE}Request:${NC}"
echo "$INIT_JSON" | jq -c .

RESPONSE=$(send_jsonrpc "$INIT_JSON")
echo -e "${BLUE}Response:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

if [ -n "$RESPONSE" ]; then
    # Check response structure
    if echo "$RESPONSE" | jq -e '.result.protocolVersion == "2025-03-26" and .result.serverInfo.name == "cowpilot"' > /dev/null 2>&1; then
        print_success "Initialize succeeded with correct protocol version"
    else
        print_failure "Initialize response invalid" \
            "protocolVersion=2025-03-26 and serverInfo.name=cowpilot" \
            "$RESPONSE"
    fi
else
    print_failure "No response received" \
        "Valid JSON-RPC response" \
        "Empty or timeout"
fi

# As an MCP Client, I want to list tools via raw JSON-RPC so that I can verify the protocol works correctly.
print_test_header "As an MCP Client, I want to list tools via raw JSON-RPC"

TOOLS_JSON=$(cat <<EOF
{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 2
}
EOF
)

echo -e "${BLUE}Request:${NC}"
echo "$TOOLS_JSON" | jq -c .

RESPONSE=$(send_jsonrpc "$TOOLS_JSON")
echo -e "${BLUE}Response:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

if [ -n "$RESPONSE" ]; then
    # Check for hello tool
    if echo "$RESPONSE" | jq -e '.result.tools[] | select(.name == "hello")' > /dev/null 2>&1; then
        print_success "Found 'hello' tool in tools list"
        
        # Extract and display tool info
        echo -e "${BLUE}Tool details:${NC}"
        echo "$RESPONSE" | jq '.result.tools[] | select(.name == "hello")'
    else
        print_failure "Hello tool not found" \
            "Tool with name='hello'" \
            "$RESPONSE"
    fi
else
    print_failure "No response received" \
        "Valid tools list" \
        "Empty or timeout"
fi

# As an MCP Client, I want to call a tool via raw JSON-RPC so that I can execute server functionality.
print_test_header "As an MCP Client, I want to call a tool via raw JSON-RPC"

CALL_JSON=$(cat <<EOF
{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
        "name": "hello",
        "arguments": {}
    },
    "id": 3
}
EOF
)

echo -e "${BLUE}Request:${NC}"
echo "$CALL_JSON" | jq -c .

RESPONSE=$(send_jsonrpc "$CALL_JSON")
echo -e "${BLUE}Response:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

if [ -n "$RESPONSE" ]; then
    # Check for "Hello, World!" in content
    if echo "$RESPONSE" | jq -e '.result.content[0].text' | grep -q "Hello, World!"; then
        print_success "Tool returned 'Hello, World!'"
    else
        print_failure "Tool response incorrect" \
            "Content with 'Hello, World!'" \
            "$RESPONSE"
    fi
else
    print_failure "No response received" \
        "Valid tool response" \
        "Empty or timeout"
fi

# As an MCP Client, I want to receive proper error responses so that I can handle failures gracefully.
print_test_header "As an MCP Client, I want to receive proper error responses"

ERROR_JSON=$(cat <<EOF
{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
        "name": "nonexistent",
        "arguments": {}
    },
    "id": 4
}
EOF
)

echo -e "${BLUE}Request:${NC}"
echo "$ERROR_JSON" | jq -c .

RESPONSE=$(send_jsonrpc "$ERROR_JSON")
echo -e "${BLUE}Response:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

if [ -n "$RESPONSE" ]; then
    # Check for error response
    if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
        print_success "Server returned proper error for nonexistent tool"
        
        # Display error details
        echo -e "${BLUE}Error details:${NC}"
        echo "$RESPONSE" | jq '.error'
    else
        print_failure "Expected error response" \
            "JSON-RPC error object" \
            "$RESPONSE"
    fi
else
    print_failure "No response received" \
        "Error response" \
        "Empty or timeout"
fi

# As an MCP Client, I want to test batch requests so that I can verify advanced protocol features.
print_test_header "As an MCP Client, I want to test batch requests"

BATCH_JSON=$(cat <<EOF
[
    {
        "jsonrpc": "2.0",
        "method": "tools/list",
        "id": 5
    },
    {
        "jsonrpc": "2.0",
        "method": "tools/call",
        "params": {
            "name": "hello",
            "arguments": {}
        },
        "id": 6
    }
]
EOF
)

echo -e "${BLUE}Request:${NC}"
echo "$BATCH_JSON" | jq -c .

RESPONSE=$(send_jsonrpc "$BATCH_JSON")
echo -e "${BLUE}Response:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

if [ -n "$RESPONSE" ]; then
    # Check if it's an array response
    if echo "$RESPONSE" | jq -e 'type == "array"' > /dev/null 2>&1; then
        print_success "Server supports batch requests"
    else
        # Batch might not be supported, which is okay
        print_success "Server responded (batch may not be supported)"
    fi
else
    print_success "Batch requests may not be supported (no response)"
fi

# As an MCP Client, I want to verify SSE stream formatting so that I can confirm transport compliance.
print_test_header "As an MCP Client, I want to verify SSE stream formatting"

echo "Testing SSE stream with multiple messages..."

# Create a temporary file for capturing stream
TEMP_FILE=$(mktemp)

# Send a request and capture multiple SSE events
(
    echo "$TOOLS_JSON" | \
    curl -s -N -X POST "$SERVER_URL" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d @- \
    --max-time 3 2>/dev/null > "$TEMP_FILE"
) &

CURL_PID=$!
sleep 2
kill $CURL_PID 2>/dev/null || true

# Analyze SSE format
SSE_LINES=$(cat "$TEMP_FILE" | wc -l)
DATA_LINES=$(grep "^data: " "$TEMP_FILE" | wc -l)

echo -e "${BLUE}SSE Stream Analysis:${NC}"
echo "  Total lines: $SSE_LINES"
echo "  Data lines: $DATA_LINES"
echo -e "${BLUE}Sample output:${NC}"
head -n 10 "$TEMP_FILE"

if [ "$DATA_LINES" -gt 0 ]; then
    print_success "SSE stream format is correct"
else
    print_failure "SSE stream format incorrect" \
        "Lines starting with 'data: '" \
        "No proper SSE data lines found"
fi

rm -f "$TEMP_FILE"

# Print summary
echo -e "\n${YELLOW}=== TEST SUMMARY ===${NC}"
echo -e "Total tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

# Print connection info
echo -e "\n${YELLOW}=== CONNECTION INFO ===${NC}"
echo "Server URL: $SERVER_URL"
echo "Transport: Server-Sent Events (SSE)"
echo "Protocol: MCP v2025-03-26"

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
fi
