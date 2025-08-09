#!/bin/bash
# Post-deploy verification for RTM OAuth

SERVER="https://rtm.fly.dev"

echo "Verifying RTM deployment OAuth compliance..."

# 1. Check WWW-Authenticate header (CRITICAL)
AUTH_HEADER=$(curl -s -I "$SERVER/mcp" | grep -i "www-authenticate")
if [ -z "$AUTH_HEADER" ]; then
    echo "❌ FAILED: Missing WWW-Authenticate header"
    echo "Claude.ai won't show Connect button!"
    exit 1
fi

# 2. Check OAuth discovery endpoints
AUTH_ENDPOINT=$(curl -s "$SERVER/.well-known/oauth-authorization-server" | jq -r '.authorization_endpoint')
if [[ "$AUTH_ENDPOINT" != *"/oauth/authorize" ]]; then
    echo "❌ FAILED: Wrong authorization endpoint: $AUTH_ENDPOINT"
    exit 1
fi

# 3. Check endpoints exist
for endpoint in "/oauth/authorize" "/oauth/token" "/rtm/callback"; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER$endpoint")
    if [ "$STATUS" = "404" ]; then
        echo "❌ FAILED: $endpoint returns 404"
        exit 1
    fi
done

echo "✅ All OAuth endpoints verified on production"
