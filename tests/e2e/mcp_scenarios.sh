#!/bin/bash

# E2E Test Suite for Cowpilot MCP Server
# Validates MCP v2025-03-26 protocol compliance using @modelcontextprotocol/inspector

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

# Test 1: Server Health Check
print_test_header "Server Health Check"

echo "Checking server health endpoint..."
HEALTH_URL="${SERVER_URL%/}/health"
if curl -s -f "$HEALTH_URL" > /dev/null 2>&1; then
    print_success "Server is healthy and responding"
else
    print_failure "Server health check failed" \
        "200 OK response from $HEALTH_URL" \
        "No response or error"
fi

# Test 2: Tool Discovery
print_test_header "Tool Discovery"

echo "Listing available tools..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method tools/list 2>&1); then
    # Check for hello tool in output
    if echo "$OUTPUT" | grep -q "hello" && \
       echo "$OUTPUT" | grep -q "Says hello to the world"; then
        print_success "Found 'hello' tool with correct description"
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

# Test 3: Tool Execution
print_test_header "Tool Execution"

echo "Calling 'hello' tool..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method tools/call --tool-name hello 2>&1); then
    # Check for "Hello, World!" in output
    if echo "$OUTPUT" | grep -q "Hello, World!"; then
        print_success "Tool executed successfully and returned 'Hello, World!'"
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

# Test 4: Error Handling - Non-existent Tool
print_test_header "Error Handling - Non-existent Tool"

echo "Calling non-existent tool..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method tools/call --tool-name nonexistent 2>&1); then
    # This should actually fail, so if we get here it's unexpected
    print_failure "Server accepted non-existent tool call" \
        "Error response for non-existent tool" \
        "$OUTPUT"
else
    # The command failed, which is expected
    if echo "$OUTPUT" | grep -qi "error\|not found\|unknown tool\|method not found"; then
        print_success "Server properly rejected non-existent tool call"
    else
        print_failure "Unexpected error response" \
            "Clear error message about non-existent tool" \
            "$OUTPUT"
    fi
fi

# Test 5: List Resources (even if empty)
print_test_header "Resource Discovery"

echo "Listing available resources..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method resources/list 2>&1); then
    print_success "Successfully queried resources endpoint"
else
    # Some servers may not implement resources
    if echo "$OUTPUT" | grep -qi "not implemented\|not supported\|method not found"; then
        print_success "Server correctly indicates resources not implemented"
    else
        print_failure "Unexpected resources list response" \
            "Successful query or clear not-implemented message" \
            "$OUTPUT"
    fi
fi

# Test 6: List Prompts (even if empty)
print_test_header "Prompt Discovery"

echo "Listing available prompts..."
if OUTPUT=$($INSPECTOR "$SERVER_URL" --method prompts/list 2>&1); then
    print_success "Successfully queried prompts endpoint"
else
    # Some servers may not implement prompts
    if echo "$OUTPUT" | grep -qi "not implemented\|not supported\|method not found"; then
        print_success "Server correctly indicates prompts not implemented"
    else
        print_failure "Unexpected prompts list response" \
            "Successful query or clear not-implemented message" \
            "$OUTPUT"
    fi
fi

# Test 7: Transport Verification
print_test_header "SSE Transport Verification"

echo "Verifying SSE transport..."
# The inspector uses SSE by default for HTTP URLs
if $INSPECTOR "$SERVER_URL" --method tools/list >/dev/null 2>&1; then
    print_success "SSE transport working correctly"
else
    print_failure "SSE transport connection failed" \
        "Successful SSE connection" \
        "Connection error"
fi

# Print summary
echo -e "\n${YELLOW}=== TEST SUMMARY ===${NC}"
echo -e "Total tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
fi
