#!/bin/bash

# Quick check if RTM is working in production

echo "RTM Production Status Check"
echo "==========================="
echo ""

APP_NAME="${1:-rtm}"
BASE_URL="https://$APP_NAME.fly.dev"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if app is responding
echo -n "1. Checking if app is online... "
if curl -s -f -o /dev/null "$BASE_URL/health" 2>/dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "   App not responding at $BASE_URL"
    echo ""
    echo "   Try:"
    echo "   - fly status -a rtm"
echo "   - fly logs -a rtm"
    exit 1
fi

# Check OAuth endpoints
echo -n "2. Checking OAuth endpoints... "
response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/oauth/authorize?client_id=test&state=test&redirect_uri=http://localhost/cb")
if [ "$response" = "200" ]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC} (HTTP $response)"
fi

# Check MCP endpoint
echo -n "3. Checking MCP endpoint... "
response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/mcp")
if [ "$response" = "401" ]; then
    echo -e "${GREEN}✓${NC} (401 - Auth required, OAuth will trigger)"
elif [ "$response" = "200" ]; then
    echo -e "${YELLOW}⚠${NC} (200 - Auth might be disabled)"
else
    echo -e "${RED}✗${NC} (HTTP $response - Unexpected)"
fi

# Check if secrets are set
echo -n "4. Checking Fly secrets... "
if command -v fly &> /dev/null; then
    if fly secrets list -a rtm 2>/dev/null | grep -q "RTM_API_KEY"; then
        echo -e "${GREEN}✓${NC}"
        echo ""
        echo "   Configured secrets:"
        fly secrets list -a rtm | grep -E "RTM_|SERVER_URL" | awk '{print "   - " $1}'
    else
        echo -e "${YELLOW}⚠${NC} Cannot verify (fly CLI issue or not logged in)"
    fi
else
    echo -e "${YELLOW}⚠${NC} (fly CLI not installed)"
fi

echo ""
echo "Summary"
echo "-------"
echo "App URL: $BASE_URL"
echo "MCP URL: $BASE_URL/mcp"
echo "OAuth: $BASE_URL/oauth/authorize"
echo ""
echo "To connect Claude Desktop:"
echo "1. Add this URL to Claude's settings: $BASE_URL/mcp"
echo "2. Click 'Connect' when prompted"
echo "3. Follow the RTM authorization flow"
echo ""
echo "To view logs:"
echo "  fly logs -a rtm"
echo ""
echo "To check detailed status:"
echo "  make -f Makefile.rtm diagnose-production"
