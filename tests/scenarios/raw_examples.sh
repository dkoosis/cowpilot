#!/bin/bash

# Direct SSE/JSON-RPC Examples for mcp adapters
# Based on https://blog.fka.dev/blog/2025-03-25-inspecting-mcp-servers-using-cli/

SERVER="https://mcp-adapters.fly.dev/"

echo "Direct MCP Server Testing using curl and jq"
echo "==========================================="
echo ""
echo "These examples show raw JSON-RPC over SSE communication:"
echo ""

echo "1. Initialize the connection:"
echo ""
echo 'echo '"'"'{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"curl-test","version":"1.0.0"}},"id":1}'"'"' | \'
echo "  curl -s -N -X POST $SERVER \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Accept: text/event-stream' \\"
echo "    -d @- | grep '^data: ' | sed 's/^data: //' | jq ."
echo ""

echo "2. List available tools:"
echo ""
echo 'echo '"'"'{"jsonrpc":"2.0","method":"tools/list","id":2}'"'"' | \'
echo "  curl -s -N -X POST $SERVER \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Accept: text/event-stream' \\"
echo "    -d @- | grep '^data: ' | sed 's/^data: //' | jq ."
echo ""

echo "3. Call the hello tool:"
echo ""
echo 'echo '"'"'{"jsonrpc":"2.0","method":"tools/call","params":{"name":"hello","arguments":{}},"id":3}'"'"' | \'
echo "  curl -s -N -X POST $SERVER \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Accept: text/event-stream' \\"
echo "    -d @- | grep '^data: ' | sed 's/^data: //' | jq ."
echo ""

echo "4. Test error handling:"
echo ""
echo 'echo '"'"'{"jsonrpc":"2.0","method":"tools/call","params":{"name":"nonexistent","arguments":{}},"id":4}'"'"' | \'
echo "  curl -s -N -X POST $SERVER \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Accept: text/event-stream' \\"
echo "    -d @- | grep '^data: ' | sed 's/^data: //' | jq ."
echo ""

echo "5. View raw SSE stream (first 20 lines):"
echo ""
echo 'echo '"'"'{"jsonrpc":"2.0","method":"tools/list","id":5}'"'"' | \'
echo "  curl -s -N -X POST $SERVER \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Accept: text/event-stream' \\"
echo "    -d @- | head -20"
echo ""

echo "Note: SSE format has each JSON response prefixed with 'data: '"
echo "The grep and sed commands extract just the JSON payload for jq to parse."
