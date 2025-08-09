#!/bin/bash

echo "RTM Quick Diagnostics"
echo "===================="
echo ""

# Check if fly CLI is available
if ! command -v fly &> /dev/null; then
    echo "❌ Fly CLI not installed"
    echo "Install with: brew install flyctl"
    exit 1
fi

# Check app status
echo "1. Checking Fly.io app status..."
fly status -a rtm-mcp 2>&1 | head -20 || fly status -a rtm 2>&1 | head -20

echo ""
echo "2. Checking recent logs for errors..."
fly logs -a rtm-mcp 2>&1 | tail -10 || fly logs -a rtm 2>&1 | tail -10

echo ""
echo "3. Checking configured secrets..."
fly secrets list -a rtm-mcp 2>&1 | grep -E "RTM_|SERVER" || fly secrets list -a rtm 2>&1 | grep -E "RTM_|SERVER"

echo ""
echo "4. Testing simple curl (with 5s timeout)..."
curl -m 5 -s -o /dev/null -w "HTTP Status: %{http_code}\n" https://rtm-mcp.fly.dev/health 2>&1 || echo "Connection failed/timeout"

echo ""
echo "5. Checking if app exists..."
fly apps list | grep -E "rtm|mcp" || echo "No RTM apps found"
