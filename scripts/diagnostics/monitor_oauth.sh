#!/bin/bash

# OAuth Flow Monitor for RTM MCP Server
# This script helps diagnose OAuth connection issues with Claude

set -e

echo "======================================"
echo "RTM OAuth Flow Diagnostic Monitor"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running locally or against production
if [ "$1" == "local" ]; then
    SERVER_URL="http://localhost:8081"
    echo -e "${YELLOW}Testing LOCAL server at $SERVER_URL${NC}"
else
    SERVER_URL="https://rtm.fly.dev"
    echo -e "${YELLOW}Testing PRODUCTION server at $SERVER_URL${NC}"
fi

echo ""
echo "1. Checking OAuth Discovery Endpoints"
echo "--------------------------------------"

# Check protected resource metadata
echo -n "Checking /.well-known/oauth-protected-resource ... "
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/.well-known/oauth-protected-resource" 2>/dev/null)
HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✓ OK (200)${NC}"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo -e "${RED}✗ Failed (HTTP $HTTP_CODE)${NC}"
    echo "$BODY"
fi

echo ""

# Check auth server metadata
echo -n "Checking /.well-known/oauth-authorization-server ... "
RESPONSE=$(curl -s -w "\n%{http_code}" "$SERVER_URL/.well-known/oauth-authorization-server" 2>/dev/null)
HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✓ OK (200)${NC}"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
else
    echo -e "${RED}✗ Failed (HTTP $HTTP_CODE)${NC}"
    echo "$BODY"
fi

echo ""
echo "2. Testing MCP Endpoint Authentication"
echo "--------------------------------------"

# Test without auth header
echo -n "Testing /mcp without auth ... "
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}' \
    -w "\nHTTP_CODE:%{http_code}\nWWW_AUTH:%{header_www-authenticate}" \
    "$SERVER_URL/mcp" 2>/dev/null)

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
WWW_AUTH=$(echo "$RESPONSE" | grep "WWW_AUTH:" | cut -d':' -f2-)
BODY=$(echo "$RESPONSE" | sed '/HTTP_CODE:/,$d')

if [ "$HTTP_CODE" == "401" ]; then
    echo -e "${GREEN}✓ Returns 401 as expected${NC}"
    if [ ! -z "$WWW_AUTH" ] && [ "$WWW_AUTH" != "WWW_AUTH:" ]; then
        echo -e "${GREEN}✓ WWW-Authenticate header present: $WWW_AUTH${NC}"
    else
        echo -e "${RED}✗ Missing WWW-Authenticate header!${NC}"
    fi
else
    echo -e "${RED}✗ Unexpected status code: $HTTP_CODE (expected 401)${NC}"
fi

echo ""
echo "3. Monitoring Production Logs (if available)"
echo "--------------------------------------------"

if [ "$1" != "local" ]; then
    echo "Fetching recent logs from Fly.io..."
    echo "(Make sure you're logged into fly CLI)"
    echo ""
    
    # Try to fetch logs
    if command -v flyctl &> /dev/null; then
        echo "Recent OAuth-related logs:"
        flyctl logs --app rtm --tail | head -50 | grep -E "(OAuth|oauth|auth|token|callback)" || echo "No OAuth-related logs found"
    else
        echo -e "${YELLOW}flyctl not installed. Install it to view production logs.${NC}"
    fi
fi

echo ""
echo "4. Browser-Side Diagnostics"
echo "---------------------------"
echo ""
echo "To diagnose from Claude's side:"
echo "1. Open Chrome/Firefox Developer Tools (F12)"
echo "2. Go to Network tab"
echo "3. Try to connect to RTM in Claude"
echo "4. Look for these requests:"
echo "   - OAuth authorization redirect"
echo "   - Callback URL with code parameter"
echo "   - Any failed requests (shown in red)"
echo ""
echo "5. In Console tab, run this to check for errors:"
echo "   > console.log(window.location.href)"
echo "   > console.log(document.cookie)"
echo ""

echo "5. Manual OAuth Flow Test"
echo "-------------------------"
echo ""
echo "Test the OAuth flow manually:"
echo ""
echo "1. Open this URL in your browser:"
echo "   $SERVER_URL/oauth/authorize?client_id=claude&redirect_uri=https://claude.ai/oauth/callback&state=test123"
echo ""
echo "2. Enter an RTM API key when prompted"
echo ""
echo "3. Check if you're redirected to:"
echo "   https://claude.ai/oauth/callback?code=XXX&state=test123"
echo ""
echo "4. Note any errors or unexpected behavior"
echo ""

if [ "$1" == "local" ]; then
    echo "6. Local Server Debugging"
    echo "-------------------------"
    echo ""
    echo "Make sure your local server is running with debug enabled:"
    echo "  DEBUG=true go run cmd/rtm/main.go"
    echo ""
    echo "Check for debug.db file:"
    if [ -f "debug.db" ]; then
        echo -e "${GREEN}✓ debug.db found${NC}"
        echo "Recent tokens:"
        sqlite3 debug.db "SELECT datetime(created_at, 'localtime') as time, substr(token, 1, 10) || '...' as token_preview FROM tokens ORDER BY created_at DESC LIMIT 5;" 2>/dev/null || echo "Could not read debug.db"
    else
        echo -e "${YELLOW}No debug.db file found${NC}"
    fi
fi
