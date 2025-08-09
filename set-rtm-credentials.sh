#!/bin/bash
# Set RTM API credentials on Fly.io

echo "Setting RTM API credentials..."

# Check if env vars exist locally
if [ -z "$RTM_API_KEY" ] || [ -z "$RTM_API_SECRET" ]; then
    echo "❌ RTM_API_KEY and RTM_API_SECRET must be set"
    echo ""
    echo "Get them from: https://www.rememberthemilk.com/services/api/keys.rtm"
    echo "Then run:"
    echo "  export RTM_API_KEY='your-key-here'"
    echo "  export RTM_API_SECRET='your-secret-here'"
    exit 1
fi

# Set on Fly.io
flyctl secrets set RTM_API_KEY="$RTM_API_KEY" RTM_API_SECRET="$RTM_API_SECRET" --app rtm

echo "Waiting for restart..."
sleep 10

# Verify
echo "Testing OAuth endpoints..."
curl -I https://rtm.fly.dev/mcp | grep -i www-authenticate
