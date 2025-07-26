#!/bin/bash
# Test SSE transport for Claude.ai integration

set -e

URL="${1:-http://localhost:8080/mcp}"
echo "Testing SSE at: $URL"

# Test 1: Initialize with SSE
echo -e "\n1. Testing SSE initialize:"
curl -X POST "$URL" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"sse-test","version":"1.0"}}}' \
  --max-time 5 -N

# Test 2: List tools with SSE
echo -e "\n\n2. Testing SSE tools/list:"
curl -X POST "$URL" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":2}' \
  --max-time 5 -N

# Test 3: Call tool with SSE
echo -e "\n\n3. Testing SSE tool call:"
curl -X POST "$URL" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","id":3,"params":{"name":"hello"}}' \
  --max-time 5 -N

echo -e "\n\nSSE tests complete"
