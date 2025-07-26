#!/bin/bash

# E2E Test Suite for Cowpilot MCP Server
# Uses official @modelcontextprotocol/inspector in CLI mode

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Server URL (default to production if not provided)
SERVER_URL="${1:-https://cowpilot.fly.dev/}"

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Inspector command base
INSPECTOR="npx @modelcontextprotocol/inspector --cli"

# Function to print test header
print_test_header() {
    echo -e "\n${YELLOW}=== TEST: $1 ==="
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    PASSED_TESTS=$((PASSED_TESTS + 1))
}

# Function to print failure
print_failure() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    echo -e "${RED}Expected:${NC} $2"
    echo -e "${RED}Actual:${NC} $3"
    FAILED_TESTS=$((FAILED_TESTS + 1))
}

# As an MCP Client, I want to verify server health so that I know the server is operational.
print_test_header "As an MCP Client, I want to verify server health"

echo "Checking server health endpoint..."
HEALTH_URL="${SERVER_URL%/}/health"
if curl -s -f "$HEALTH_URL" > /dev/null 2>&1; then
    print_success "Server is healthy and responding"
else
    print_failure "Server health check failed" \
        "200 OK response from $HEALTH_URL" \
        "No response or error"
fi

# As an MCP Client, I want to list available tools so that I know what capabilities the server offers.
print_test_header "As an MCP Client, I want to list available tools"

echo "Listing available tools..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method tools/list 2>&1); then
    # Check for hello tool in output
    if echo "$OUTPUT" | grep -q '"name":\s*"hello"' && \
       echo "$OUTPUT" | grep -q '"description":\s*"Says hello to the world"'; then
        print_success "Found 'hello' tool with correct description"
        echo "Tools response: $OUTPUT"
    else
        print_failure "Tool list doesn't contain expected 'hello' tool" \
            "Tool named 'hello' with description" \
            "$OUTPUT"
    fi
else
    print_failure "Failed to list tools" \
        "Successful tool listing" \
        "$OUTPUT"
fi

# As an MCP Client, I want to call a tool so that I can execute server-side functionality.
print_test_header "As an MCP Client, I want to call the 'hello' tool"

echo "Calling 'hello' tool..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method tools/call --tool-name hello 2>&1); then
    # Check for "Hello, World!" in output
    if echo "$OUTPUT" | grep -q "Hello, World!"; then
        print_success "Tool executed successfully and returned 'Hello, World!'"
        echo "Tool response: $OUTPUT"
    else
        print_failure "Tool call didn't return expected output" \
            "Output containing 'Hello, World!'" \
            "$OUTPUT"
    fi
else
    print_failure "Failed to call hello tool" \
        "Successful tool execution" \
        "$OUTPUT"
fi

# As an MCP Client, I want to receive a clear error when calling a non-existent tool so that I can handle the failure gracefully.
print_test_header "As an MCP Client, I want to receive an error for a non-existent tool"

echo "Calling non-existent tool..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method tools/call --tool-name nonexistent 2>&1); then
    # If it succeeds, that's wrong
    print_failure "Server accepted non-existent tool call" \
        "Error response for non-existent tool" \
        "$OUTPUT"
else
    # The command should fail - check for error in output
    if echo "$OUTPUT" | grep -qi "error\|not found\|unknown tool\|-32602"; then
        print_success "Server properly rejected non-existent tool call"
        echo "Error response: $OUTPUT"
    else
        print_failure "Unexpected error response" \
            "Clear error message about non-existent tool" \
            "$OUTPUT"
    fi
fi

# As an MCP Client, I want to query the resources endpoint so that I can discover available data sources.
print_test_header "As an MCP Client, I want to query the resources endpoint"

echo "Listing available resources..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method resources/list 2>&1); then
    # Success - resources endpoint works (even if empty)
    print_success "Successfully queried resources endpoint"
    echo "Resources response: $OUTPUT"
else
    # Check if it's a "not implemented" error
    if echo "$OUTPUT" | grep -qi "not implemented\|not supported\|method not found\|-32601"; then
        print_success "Server correctly indicates resources not implemented"
    else
        print_failure "Unexpected resources list response" \
            "Successful query or clear not-implemented message" \
            "$OUTPUT"
    fi
fi

# Print summary
echo -e "\n${YELLOW}=== TEST SUMMARY ==="
echo -e "Total tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS"
echo -e "${RED}Failed: $FAILED_TESTS"

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed!"
    exit 0
else
    echo -e "\n${RED}Some tests failed!"
    exit 1
fi
