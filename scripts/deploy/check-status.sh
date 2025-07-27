#!/bin/bash
# Quick status check for Cowpilot deployment
# Run this to verify everything is ready for Claude.ai registration

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

APP_NAME="cowpilot"
BASE_URL="https://$APP_NAME.fly.dev"

echo -e "${CYAN}🔍 Cowpilot Status Check${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

TOTAL_CHECKS=0
PASSED_CHECKS=0

# Function to check endpoint
check_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"
    
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    echo -ne "${BLUE}Checking $name... ${NC}"
    
    HTTP_STATUS=$(curl -s -w "%{http_code}" -o /tmp/response.txt "$url")
    
    if [ "$HTTP_STATUS" = "$expected_status" ]; then
        echo -e "${GREEN}✓ ($HTTP_STATUS)${NC}"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        if [ -s /tmp/response.txt ] && command -v jq &> /dev/null; then
            cat /tmp/response.txt | jq -C '.' 2>/dev/null || cat /tmp/response.txt
        fi
        return 0
    else
        echo -e "${RED}✗ (Expected $expected_status, got $HTTP_STATUS)${NC}"
        return 1
    fi
}

# Check health
check_endpoint "Health endpoint" "$BASE_URL/health"

# Check OAuth metadata
echo -e "\n${YELLOW}OAuth Discovery:${NC}"
check_endpoint "Protected resource metadata" \
    "$BASE_URL/.well-known/oauth-protected-resource"
check_endpoint "Authorization server metadata" \
    "$BASE_URL/.well-known/oauth-authorization-server"

# Check auth requirement
echo -e "\n${YELLOW}Authentication:${NC}"
check_endpoint "MCP endpoint (should require auth)" \
    "$BASE_URL/mcp" \
    "401"

# Check CORS headers
echo -e "\n${YELLOW}CORS Configuration:${NC}"
echo -ne "${BLUE}Checking CORS headers... ${NC}"
CORS_HEADERS=$(curl -s -I -X OPTIONS "$BASE_URL/mcp" \
    -H "Origin: https://claude.ai" \
    -H "Access-Control-Request-Method: POST" 2>/dev/null | grep -i "access-control")

if echo "$CORS_HEADERS" | grep -q "claude.ai"; then
    echo -e "${GREEN}✓${NC}"
    echo "$CORS_HEADERS" | sed 's/^/  /'
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
else
    echo -e "${RED}✗ CORS not configured for claude.ai${NC}"
fi
TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

# Check debug mode
echo -e "\n${YELLOW}Debug Configuration:${NC}"
echo -ne "${BLUE}Checking if debug is enabled... ${NC}"
if fly secrets list -a "$APP_NAME" 2>/dev/null | grep -q "MCP_DEBUG"; then
    echo -e "${GREEN}✓ Debug mode enabled${NC}"
    fly secrets list -a "$APP_NAME" | grep MCP_ | sed 's/^/  /'
else
    echo -e "${YELLOW}⚠️  Debug mode not enabled${NC}"
fi

# Summary
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}Summary:${NC} $PASSED_CHECKS/$TOTAL_CHECKS checks passed"

if [ $PASSED_CHECKS -eq $TOTAL_CHECKS ]; then
    echo -e "${GREEN}✓ All systems ready for Claude.ai registration!${NC}"
    echo -e "\n${YELLOW}Registration info:${NC}"
    echo -e "  Name:        ${CYAN}Cowpilot Tools${NC}"
    echo -e "  Description: ${CYAN}MCP server providing various utility tools including echo, time, base64 encoding and more${NC}"
    echo -e "  URL:         ${CYAN}$BASE_URL${NC}"
else
    echo -e "${RED}✗ Some checks failed. Review the output above.${NC}"
    echo -e "${YELLOW}Run 'fly logs -a $APP_NAME' to investigate${NC}"
fi
