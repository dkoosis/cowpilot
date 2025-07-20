#!/bin/bash
# Quick E2E test without browser

echo "Testing cowpilot server..."

# Test production server
SERVER="https://cowpilot.fly.dev/"

echo "1. Health check..."
curl -s "${SERVER}health" | grep -q "OK" && echo "✓ Health check passed" || echo "✗ Health check failed"

echo -e "\n2. Testing hello tool with raw JSON-RPC..."
RESPONSE=$(echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"hello","arguments":{}},"id":1}' | \
  curl -s -N -X POST "$SERVER" \
    -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    -d @- | grep '^data: ' | sed 's/^data: //' | head -n 1)

echo "Response: $RESPONSE"

if echo "$RESPONSE" | jq -e '.result.content[0].text' | grep -q "Hello, World!"; then
    echo "✓ Hello tool test passed"
else
    echo "✗ Hello tool test failed"
    exit 1
fi

echo -e "\n✓ All tests passed!"
