#!/bin/bash
# Build and test cowpilot

echo "=== Building Cowpilot ==="
make build

echo -e "\n=== Running Tests ==="
make test

echo -e "\n=== Running Local Server Test ==="
# Start server in background
./bin/cowpilot &
SERVER_PID=$!
sleep 2

# Test with curl
echo "Testing hello tool..."
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"hello","arguments":{}},"id":1}' | \
  curl -s -X POST http://localhost:8080 \
    -H "Content-Type: application/json" \
    -d @- | jq .

# Kill server
kill $SERVER_PID 2>/dev/null

echo -e "\n=== Testing Production ==="
make e2e-test-prod
