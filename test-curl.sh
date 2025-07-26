#!/bin/bash
cd /Users/vcto/Projects/cowpilot

# Check if jq is available, if not, use cat
if ! command -v jq &> /dev/null; then
    JQ="cat"
else
    JQ="jq ."
fi

# Build and start server
go build -o ./bin/cowpilot ./cmd/cowpilot
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

echo "=== Testing with curl directly ==="

# Test 1: Initialize
echo -e "\n1. Initialize:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' \
  http://localhost:8080/ | $JQ

# Test 2: List tools  
echo -e "\n2. List tools:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
  http://localhost:8080/ | $JQ

# Test 3: Call hello tool
echo -e "\n3. Call hello tool:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"hello"}}' \
  http://localhost:8080/ | $JQ

# Test 4: Call echo tool with args
echo -e "\n4. Call echo tool with args:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{"message":"test"}}}' \
  http://localhost:8080/ | $JQ

kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null
