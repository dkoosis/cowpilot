#!/bin/bash
# Test OAuth callback flow

set -e

echo "=== OAuth Callback Test ==="

# 1. Start the cowpilot server
echo "Building cowpilot..."
go build -o bin/cowpilot-test cmd/cowpilot/main.go

echo "Starting server..."
./bin/cowpilot-test &
SERVER_PID=$!
sleep 2

# 2. Test OAuth metadata endpoints
echo -e "\n--- Testing OAuth metadata ---"
curl -s http://localhost:8080/.well-known/oauth-protected-resource | jq .
curl -s http://localhost:8080/.well-known/oauth-authorization-server | jq .

# 3. Test authorize endpoint
echo -e "\n--- Testing authorize endpoint ---"
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8080/oauth/authorize?client_id=test-client&redirect_uri=http://localhost:9090/callback&state=test-state)
STATUS_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$STATUS_CODE" == "200" ]; then
    echo "✓ Authorize endpoint returned 200"
    if echo "$BODY" | grep -q "Connect Remember The Milk"; then
        echo "✓ Form page rendered correctly"
    fi
else
    echo "✗ Authorize endpoint failed with status $STATUS_CODE"
fi

# 4. Test CSRF token generation
echo -e "\n--- Testing CSRF protection ---"
# Extract CSRF token from form
CSRF_TOKEN=$(echo "$BODY" | grep -oP 'name="csrf_state" value="\K[^"]+')
if [ -n "$CSRF_TOKEN" ]; then
    echo "✓ CSRF token generated: ${CSRF_TOKEN:0:8}..."
else
    echo "✗ Failed to extract CSRF token"
fi

# 5. Cleanup
echo -e "\n--- Cleanup ---"
kill $SERVER_PID 2>/dev/null || true
rm -f bin/cowpilot-test

echo -e "\n=== Test Complete ==="
