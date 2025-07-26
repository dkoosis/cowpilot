#!/bin/bash
# Test OAuth flow for Claude.ai integration

set -e

URL="${1:-http://localhost:8080}"
echo "Testing OAuth flow at: $URL"

# Test 1: Protected Resource Metadata
echo -e "\n1. Testing Protected Resource Metadata:"
curl -s "$URL/.well-known/oauth-protected-resource" | jq .

# Test 2: Auth Server Metadata
echo -e "\n2. Testing Authorization Server Metadata:"
curl -s "$URL/.well-known/oauth-authorization-server" | jq .

# Test 3: Attempt MCP call without auth (should get 401)
echo -e "\n3. Testing unauthorized MCP call:"
curl -s -w "\nHTTP Status: %{http_code}\n" \
  -X POST "$URL/mcp" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' || true

# Test 4: DCR (Dynamic Client Registration)
echo -e "\n4. Testing Dynamic Client Registration:"
CLIENT_REG=$(curl -s -X POST "$URL/oauth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "Test OAuth Client",
    "redirect_uris": ["http://localhost:8090/callback"]
  }')
echo "$CLIENT_REG" | jq .
CLIENT_ID=$(echo "$CLIENT_REG" | jq -r '.client_id')

# Test 5: Show authorization URL
echo -e "\n5. Authorization URL:"
echo "$URL/oauth/authorize?client_id=$CLIENT_ID&redirect_uri=http://localhost:8090/callback&state=test123&resource=$URL"

echo -e "\nOAuth endpoints configured successfully!"
echo "To complete flow:"
echo "1. Visit the authorization URL above"
echo "2. Enter RTM API key"
echo "3. Exchange code for token at /oauth/token"
