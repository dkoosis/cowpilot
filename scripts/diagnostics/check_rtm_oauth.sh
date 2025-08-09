#!/bin/bash
set -e

# RTM OAuth Registration Diagnostic
# Checks critical points for Claude AI registration

SERVER=${1:-"https://rtm.fly.dev"}
echo "Testing: $SERVER"
echo "================================"

# 1. Check discovery endpoints return correct paths
echo -e "\n1. OAuth Discovery Endpoints:"
echo -n "  Protected Resource: "
curl -s "$SERVER/.well-known/oauth-protected-resource" | jq -r '.authorization_servers[0]' 2>/dev/null || echo "FAILED"

echo -n "  Auth Server - authorize: "
curl -s "$SERVER/.well-known/oauth-authorization-server" | jq -r '.authorization_endpoint' 2>/dev/null || echo "FAILED"

echo -n "  Auth Server - token: "
curl -s "$SERVER/.well-known/oauth-authorization-server" | jq -r '.token_endpoint' 2>/dev/null || echo "FAILED"

# 2. Check 401 returns WWW-Authenticate header
echo -e "\n2. WWW-Authenticate Header Check:"
AUTH_HEADER=$(curl -s -I "$SERVER/mcp" | grep -i "www-authenticate" | head -1)
if [ -n "$AUTH_HEADER" ]; then
    echo "  ✓ $AUTH_HEADER"
else
    echo "  ✗ MISSING - Claude.ai won't show Connect button!"
fi

# 3. Check authorization endpoint accessibility
echo -e "\n3. Authorization Endpoint:"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER/oauth/authorize?client_id=test")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "302" ] || [ "$STATUS" = "400" ]; then
    echo "  ✓ Responding (HTTP $STATUS)"
else
    echo "  ✗ Not accessible (HTTP $STATUS)"
fi

# 4. Check token endpoint
echo -e "\n4. Token Endpoint:"
TOKEN_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER/oauth/token" -d "grant_type=authorization_code&code=test")
if [ "$TOKEN_STATUS" != "404" ]; then
    echo "  ✓ Endpoint exists (HTTP $TOKEN_STATUS)"
else
    echo "  ✗ Not found (HTTP $TOKEN_STATUS)"
fi

# 5. Check if server is healthy
echo -e "\n5. Server Health:"
HEALTH=$(curl -s "$SERVER/health")
if [ "$HEALTH" = "OK" ]; then
    echo "  ✓ Server is healthy"
else
    echo "  ✗ Server not responding correctly"
fi

echo -e "\n================================"
echo "Key Issues to Fix:"
echo "1. Endpoints in discovery MUST match actual routes (/oauth/ prefix)"
echo "2. WWW-Authenticate header MUST be present on 401 responses"
echo "3. Token storage must persist across requests"
