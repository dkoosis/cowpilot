#!/bin/bash

# Fix RTM registration issues with Claude AI

echo "Fixing RTM MCP Registration..."

# 1. Rebuild with latest fixes
cd /Users/vcto/Projects/cowpilot
go build -o bin/rtm cmd/rtm/main.go

# 2. Deploy to Fly.io
flyctl deploy --config fly-rtm.toml --app rtm

# 3. Verify critical endpoints
sleep 5
echo -e "\nVerifying deployment..."

# Check WWW-Authenticate header (CRITICAL for Claude)
if curl -s -I https://rtm.fly.dev/mcp | grep -q "WWW-Authenticate"; then
    echo "✓ WWW-Authenticate header present"
else
    echo "✗ MISSING WWW-Authenticate - Claude won't show Connect button!"
    exit 1
fi

# Verify OAuth endpoints match discovery
AUTH_EP=$(curl -s https://rtm.fly.dev/.well-known/oauth-authorization-server | jq -r '.authorization_endpoint')
if [[ "$AUTH_EP" == *"/oauth/authorize"* ]]; then
    echo "✓ OAuth endpoints correctly advertised"
else
    echo "✗ OAuth endpoint mismatch: $AUTH_EP"
    exit 1
fi

echo -e "\n✓ RTM server should now register with Claude AI"
echo "Test at: https://claude.ai/settings/connections"
