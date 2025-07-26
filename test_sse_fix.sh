#!/bin/bash
# Quick test of the SSE fix
set -e

echo "Building server..."
cd /Users/vcto/Projects/cowpilot
make build

echo "Starting SSE server..."
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

echo "Testing SSE connection..."
timeout 5 curl -s -N -X POST http://localhost:8080/ \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}},"id":1}' \
  | head -n 5

echo -e "\nTesting MCP Inspector..."
timeout 10 npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --method tools/list

echo -e "\nStopping server..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo "Test complete!"
