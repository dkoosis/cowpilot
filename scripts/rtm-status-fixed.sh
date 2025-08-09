#!/bin/bash

# RTM Quick Status Check - CORRECTED APP NAME

echo "RTM Production Status Check"
echo "==========================="
echo ""

# The app is named "rtm" not "rtm-mcp"
APP_NAME="rtm"
BASE_URL="https://rtm.fly.dev"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Quick check if fly CLI works
if ! command -v fly &> /dev/null; then
    echo "❌ Fly CLI not installed. Install with: brew install flyctl"
    exit 1
fi

# Check app exists
echo -n "1. Checking if app 'rtm' exists... "
if fly apps list 2>/dev/null | grep -q "^rtm "; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo ""
    echo "   App 'rtm' not found. You need to deploy it first:"
    echo "   make deploy-rtm"
    exit 1
fi

# Check app status
echo -n "2. Checking app status... "
status=$(fly status -a rtm --json 2>/dev/null | grep -o '"Status":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ "$status" = "running" ]; then
    echo -e "${GREEN}✓${NC} ($status)"
elif [ -n "$status" ]; then
    echo -e "${YELLOW}⚠${NC} ($status)"
    echo "   Try: fly restart -a rtm"
else
    echo -e "${RED}✗${NC} (unknown)"
fi

# Check HTTP endpoint with timeout
echo -n "3. Checking HTTPS endpoint... "
response=$(curl -m 5 -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" 2>/dev/null)
if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓${NC} (HTTP $response)"
else
    echo -e "${RED}✗${NC} (HTTP $response or timeout)"
    echo "   The app may be suspended. Try:"
    echo "   fly scale count 1 -a rtm"
    echo "   fly restart -a rtm"
fi

# Check secrets
echo -n "4. Checking RTM secrets... "
if fly secrets list -a rtm 2>/dev/null | grep -q "RTM_API_KEY"; then
    echo -e "${GREEN}✓${NC}"
    echo "   Configured secrets:"
    fly secrets list -a rtm 2>/dev/null | grep -E "RTM_|SERVER_URL" | awk '{print "   - " $1}'
else
    echo -e "${RED}✗${NC}"
    echo "   Missing RTM secrets! Set them with:"
    echo "   fly secrets set RTM_API_KEY=your_key RTM_API_SECRET=your_secret -a rtm"
fi

echo ""
echo "Summary"
echo "-------"
echo "App Name: rtm"
echo "URL: $BASE_URL"
echo "MCP URL: $BASE_URL/mcp"
echo "OAuth: $BASE_URL/oauth/authorize"
echo ""

# Quick fix suggestions
if [ "$response" != "200" ]; then
    echo "🔧 Quick Fix Commands:"
    echo "   fly restart -a rtm          # Restart the app"
    echo "   fly scale count 1 -a rtm    # Ensure 1 instance running"
    echo "   fly logs -a rtm             # Check logs for errors"
    echo "   fly deploy -a rtm -c fly-rtm.toml  # Redeploy"
fi
