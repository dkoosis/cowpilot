#!/bin/bash
# Fix RTM OAuth registration with Claude.ai

echo "=== RTM OAuth Fix ==="

# 1. Check current state
echo "1. Checking current RTM server..."
RESPONSE=$(curl -s -I https://rtm.fly.dev/mcp | head -1)
echo "   Response: $RESPONSE"

WWW_AUTH=$(curl -s -I https://rtm.fly.dev/mcp | grep -i "www-authenticate" | head -1)
if [ -z "$WWW_AUTH" ]; then
    echo "   ❌ Missing WWW-Authenticate header"
    NEEDS_FIX=true
else
    echo "   ✓ WWW-Authenticate present"
fi

# 2. Check OAuth discovery
echo "2. Checking OAuth discovery..."
AUTH_EP=$(curl -s https://rtm.fly.dev/.well-known/oauth-authorization-server | jq -r '.authorization_endpoint' 2>/dev/null)
if [[ "$AUTH_EP" == *"/oauth/authorize" ]]; then
    echo "   ✓ OAuth endpoints correct: $AUTH_EP"
else
    echo "   ❌ Wrong OAuth endpoint: $AUTH_EP"
    NEEDS_FIX=true
fi

if [ "$NEEDS_FIX" = true ]; then
    echo ""
    echo "=== Fixing RTM OAuth ==="
    
    # Rebuild and deploy
    echo "3. Rebuilding RTM server..."
    cd /Users/vcto/Projects/cowpilot
    go build -o bin/rtm cmd/rtm/main.go
    
    echo "4. Deploying to Fly.io..."
    flyctl deploy --config fly-rtm.toml --app rtm
    
    echo "5. Waiting for deployment..."
    sleep 10
    
    # Verify fix
    echo "6. Verifying fix..."
    go test -v ./tests/integration -run TestRTMOAuthValidation
else
    echo ""
    echo "✓ RTM OAuth appears correct"
fi

echo ""
echo "=== Test in Claude.ai ==="
echo "1. Go to: https://claude.ai/settings/connections"
echo "2. Add MCP Server: https://rtm.fly.dev"
echo "3. Should see 'Connect' button"
