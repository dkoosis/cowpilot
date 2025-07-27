#!/bin/bash
# Monitor Cowpilot Registration Attempts on Claude.ai
# This script helps monitor OAuth flow and debug registration issues

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

APP_NAME="cowpilot"

echo -e "${CYAN}🔍 Cowpilot Registration Monitor${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Check if fly CLI is available
if ! command -v fly &> /dev/null; then
    echo -e "${RED}✗ fly CLI not found. Please install: https://fly.io/docs/flyctl/install/${NC}"
    exit 1
fi

# Quick health check
echo -e "${BLUE}Checking app health...${NC}"
if curl -s -f "https://$APP_NAME.fly.dev/health" > /dev/null; then
    echo -e "${GREEN}✓ App is healthy${NC}"
else
    echo -e "${RED}✗ App health check failed${NC}"
    echo -e "${YELLOW}Check status with: fly status -a $APP_NAME${NC}"
fi

# Show registration info
echo -e "\n${YELLOW}Claude.ai Registration Info:${NC}"
echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Name:        ${CYAN}Cowpilot Tools${NC}"
echo -e "Description: ${CYAN}MCP server providing various utility tools including echo, time, base64 encoding and more${NC}"
echo -e "URL:         ${CYAN}https://$APP_NAME.fly.dev${NC}"

# Show OAuth endpoints
echo -e "\n${YELLOW}OAuth Endpoints:${NC}"
echo -e "━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Discovery:   ${CYAN}https://$APP_NAME.fly.dev/.well-known/oauth-authorization-server${NC}"
echo -e "Authorize:   ${CYAN}https://$APP_NAME.fly.dev/oauth/authorize${NC}"
echo -e "Token:       ${CYAN}https://$APP_NAME.fly.dev/oauth/token${NC}"
echo -e "Register:    ${CYAN}https://$APP_NAME.fly.dev/oauth/register${NC}"

# Instructions
echo -e "\n${YELLOW}Monitoring Instructions:${NC}"
echo -e "━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "1. Open Claude.ai in your browser"
echo -e "2. Navigate to MCP server settings"
echo -e "3. Add the connector with the info above"
echo -e "4. Watch the logs below for registration attempts"

echo -e "\n${CYAN}Starting log monitor...${NC}"
echo -e "${BLUE}Press Ctrl+C to stop${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

# Start monitoring with helpful filters
fly logs -a "$APP_NAME" | grep -E --line-buffered \
    'DEBUG|OAuth|CORS|http_request|well-known|authorize|register|token|Bearer|401|session' | \
    while IFS= read -r line; do
        # Highlight important lines
        if echo "$line" | grep -q "well-known"; then
            echo -e "${CYAN}[DISCOVERY]${NC} $line"
        elif echo "$line" | grep -q "register"; then
            echo -e "${YELLOW}[REGISTER]${NC} $line"
        elif echo "$line" | grep -q "authorize"; then
            echo -e "${YELLOW}[AUTHORIZE]${NC} $line"
        elif echo "$line" | grep -q "token"; then
            echo -e "${YELLOW}[TOKEN]${NC} $line"
        elif echo "$line" | grep -q "401"; then
            echo -e "${RED}[UNAUTH]${NC} $line"
        elif echo "$line" | grep -q "CORS"; then
            echo -e "${BLUE}[CORS]${NC} $line"
        elif echo "$line" | grep -q "DEBUG"; then
            echo -e "${GREEN}[DEBUG]${NC} $line"
        else
            echo "$line"
        fi
    done
