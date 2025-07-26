#!/bin/bash

# Quick transport test to diagnose the issue
set -e

echo "Testing transport compatibility..."

# Start server in background
echo "Starting cowpilot server..."
cd /Users/vcto/Projects/cowpilot
make build
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "Testing different transport methods:"

# Test 1: Basic health check
echo "1. Health check:"
curl -s http://localhost:8080/health
echo ""

# Test 2: Regular HTTP POST with JSON
echo "2. HTTP POST (JSON-RPC):"
curl -s -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' | jq -c . || echo "Failed"

# Test 3: SSE connection attempt  
echo "3. SSE connection test:"
timeout 3 curl -s -N http://localhost:8080/ \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' || echo "SSE failed/timeout"

# Test 4: MCP Inspector connection
echo "4. MCP Inspector test:"
timeout 5 npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --method tools/list || echo "Inspector failed"

# Clean up
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo "Test complete."
