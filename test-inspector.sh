#!/bin/bash
cd /Users/vcto/Projects/cowpilot

# Build and start server
go build -o ./bin/cowpilot ./cmd/cowpilot
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

echo "=== Testing inspector directly (with timeouts) ==="

# Test 1: Check inspector version
echo -e "\n1. Inspector version:"
timeout 5 npx @modelcontextprotocol/inspector --version || echo "TIMEOUT"

# Test 2: Try initialize with inspector
echo -e "\n2. Initialize with inspector:"
timeout 10 npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --transport http --method initialize --params '{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}' || echo "TIMEOUT after 10s"

# Test 3: List tools with inspector
echo -e "\n3. List tools with inspector:"
timeout 10 npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --transport http --method tools/list || echo "TIMEOUT after 10s"

# Test 4: Try without --transport flag (likely to hang)
echo -e "\n4. List tools without --transport flag (5s timeout):"
timeout 5 npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --method tools/list || echo "TIMEOUT after 5s - SSE mode hanging"

# Test 5: Try with explicit HTTP endpoint
echo -e "\n5. Try with /mcp endpoint:"
timeout 10 npx @modelcontextprotocol/inspector --cli http://localhost:8080/mcp --method tools/list || echo "TIMEOUT after 10s"

# Test 6: Raw curl for comparison
echo -e "\n6. Raw curl (should work):"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
  http://localhost:8080/ | head -c 200

echo -e "\n\nShutting down server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null
