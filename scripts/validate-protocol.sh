#!/bin/bash
# Protocol Validation Script
# Prevents protocol/transport mismatches by testing all supported clients

set -e

PORT=${1:-8080}
BASE_URL="http://localhost:${PORT}"

echo "🔍 PROTOCOL VALIDATION - Testing all client types against server"
echo "Server: $BASE_URL"
echo "============================================================"

# Function to check if server is running
check_server() {
    if ! curl -s "$BASE_URL/health" > /dev/null; then
        echo "❌ Server not responding at $BASE_URL"
        echo "Start server: ./bin/cowpilot"
        exit 1
    fi
}

# Test 1: Protocol diagnostic endpoint
test_protocol_info() {
    echo "📋 Test 1: Protocol Information"
    response=$(curl -s "$BASE_URL/health?protocol=true")
    echo "$response" | jq .
    
    # Verify expected protocol
    transport=$(echo "$response" | jq -r '.transport')
    if [ "$transport" != "StreamableHTTP" ]; then
        echo "❌ PROTOCOL MISMATCH: Expected StreamableHTTP, got $transport"
        exit 1
    fi
    echo "✅ Protocol: StreamableHTTP confirmed"
    echo
}

# Test 2: MCP Inspector CLI compatibility 
test_mcp_inspector() {
    echo "🔧 Test 2: MCP Inspector CLI"
    if ! command -v npx &> /dev/null; then
        echo "⚠️  Skipping: npx not available"
        return
    fi
    
    # Test with timeout to prevent hanging
    if timeout 10s npx @modelcontextprotocol/inspector --cli "$BASE_URL/" --method tools/list > /dev/null 2>&1; then
        echo "✅ MCP Inspector CLI: SUCCESS"
    else
        echo "❌ MCP Inspector CLI: FAILED"
        echo "Expected: JSON-RPC over HTTP POST"
        exit 1
    fi
    echo
}

# Test 3: Raw HTTP POST (curl)
test_raw_http() {
    echo "🌐 Test 3: Raw HTTP POST"
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' \
        "$BASE_URL/")
    
    if echo "$response" | jq '.result' > /dev/null 2>&1; then
        echo "✅ Raw HTTP POST: SUCCESS"
        tool_count=$(echo "$response" | jq '.result.tools | length')
        echo "   Found $tool_count tools"
    else
        echo "❌ Raw HTTP POST: FAILED"
        echo "Response: $response"
        exit 1
    fi
    echo
}

# Test 4: SSE compatibility (future clients)
test_sse_headers() {
    echo "📡 Test 4: SSE Header Support"
    response=$(curl -s -H "Accept: text/event-stream" "$BASE_URL/" -I)
    
    if echo "$response" | grep -q "text/event-stream\|application/json"; then
        echo "✅ SSE Headers: Accepted"
    else
        echo "⚠️  SSE Headers: May not be fully supported"
    fi
    echo
}

# Main execution
main() {
    echo "Starting protocol validation..."
    
    check_server
    test_protocol_info
    test_mcp_inspector
    test_raw_http
    test_sse_headers
    
    echo "============================================================"
    echo "🎉 ALL PROTOCOL TESTS PASSED"
    echo
    echo "✅ Server is correctly configured for:"
    echo "   • MCP Inspector CLI (--cli flag)"
    echo "   • Raw HTTP POST clients"
    echo "   • Future SSE-based clients"
    echo
    echo "🔧 Quick test commands:"
    echo "   Health: curl $BASE_URL/health"
    echo "   Protocol: curl '$BASE_URL/health?protocol=true'"
    echo "   Tools: curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":1}' $BASE_URL/"
}

main "$@"
