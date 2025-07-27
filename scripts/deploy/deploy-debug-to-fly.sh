#!/bin/bash
# Deploy Cowpilot to Fly.io with Debug Enabled
# This script handles the complete deployment process including cleanup, build, deploy, and verification

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

APP_NAME="cowpilot"
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo -e "${CYAN}🚀 Cowpilot Debug Deployment Script${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Function to check command status
check_status() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $1${NC}"
    else
        echo -e "${RED}✗ $1 failed${NC}"
        exit 1
    fi
}

# Function to wait for user confirmation
confirm() {
    echo -e "${YELLOW}$1${NC}"
    read -p "Continue? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Deployment cancelled${NC}"
        exit 1
    fi
}

cd "$PROJECT_ROOT"

# Step 1: Kill Local Processes
echo -e "\n${CYAN}Step 1: Cleaning up local processes${NC}"
echo -e "${BLUE}Killing any running cowpilot or debug proxy processes...${NC}"

pkill -f mcp-debug-proxy 2>/dev/null || true
pkill -f cowpilot 2>/dev/null || true
lsof -ti:8080 | xargs kill -9 2>/dev/null || true
lsof -ti:8081 | xargs kill -9 2>/dev/null || true

# Verify nothing is running
if ps aux | grep -E "cowpilot|mcp-debug" | grep -v grep > /dev/null; then
    echo -e "${YELLOW}Warning: Some processes may still be running${NC}"
    ps aux | grep -E "cowpilot|mcp-debug" | grep -v grep
else
    echo -e "${GREEN}✓ All local processes cleaned up${NC}"
fi

# Step 2: Check Fly.io Status
echo -e "\n${CYAN}Step 2: Checking current Fly.io deployment${NC}"

if ! command -v fly &> /dev/null; then
    echo -e "${RED}✗ fly CLI not found. Please install: https://fly.io/docs/flyctl/install/${NC}"
    exit 1
fi

echo -e "${BLUE}Current deployment status:${NC}"
fly status -a "$APP_NAME" 2>/dev/null || {
    echo -e "${RED}✗ App '$APP_NAME' not found on Fly.io${NC}"
    echo -e "${YELLOW}Run 'fly apps create $APP_NAME' first${NC}"
    exit 1
}

# Step 3: Set Debug Environment Variables
echo -e "\n${CYAN}Step 3: Configuring debug environment variables${NC}"

confirm "This will enable debug logging on production. Are you sure?"

echo -e "${BLUE}Setting debug environment variables...${NC}"
fly secrets set -a "$APP_NAME" \
    MCP_DEBUG=true \
    MCP_DEBUG_LEVEL=INFO \
    MCP_DEBUG_STORAGE=memory \
    --stage

check_status "Environment variables set"

echo -e "${BLUE}Current secrets:${NC}"
fly secrets list -a "$APP_NAME"

# Step 4: Build
echo -e "\n${CYAN}Step 4: Building application${NC}"

# Ask if they want to run tests
echo -e "${YELLOW}Run tests before deployment?${NC}"
read -p "(y/n) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}Running tests...${NC}"
    make clean
    if make build; then
        check_status "Build with tests"
    else
        echo -e "${RED}✗ Tests failed${NC}"
        confirm "Tests failed. Deploy anyway?"
        echo -e "${BLUE}Building without tests...${NC}"
        go build -o bin/cowpilot cmd/cowpilot/main.go
        check_status "Build without tests"
    fi
else
    echo -e "${BLUE}Building without tests...${NC}"
    make clean
    go build -o bin/cowpilot cmd/cowpilot/main.go
    check_status "Build without tests"
fi

# Step 5: Deploy
echo -e "\n${CYAN}Step 5: Deploying to Fly.io${NC}"
echo -e "${BLUE}Starting deployment...${NC}"

if fly deploy -a "$APP_NAME"; then
    check_status "Deployment"
else
    echo -e "${RED}✗ Deployment failed${NC}"
    echo -e "${YELLOW}Check logs with: fly logs -a $APP_NAME${NC}"
    exit 1
fi

# Step 6: Verify Deployment
echo -e "\n${CYAN}Step 6: Verifying deployment${NC}"

echo -e "${BLUE}Waiting for app to be healthy...${NC}"
sleep 5

# Check app status
if fly status -a "$APP_NAME" | grep -q "running"; then
    echo -e "${GREEN}✓ App is running${NC}"
else
    echo -e "${RED}✗ App may not be healthy${NC}"
    echo -e "${YELLOW}Check status with: fly status -a $APP_NAME${NC}"
fi

# Test endpoints
echo -e "\n${BLUE}Testing OAuth endpoints...${NC}"

echo -e "${BLUE}Testing protected resource metadata:${NC}"
if curl -s "https://$APP_NAME.fly.dev/.well-known/oauth-protected-resource" | jq . 2>/dev/null; then
    check_status "Protected resource metadata"
else
    echo -e "${RED}✗ Protected resource metadata failed${NC}"
fi

echo -e "\n${BLUE}Testing authorization server metadata:${NC}"
if curl -s "https://$APP_NAME.fly.dev/.well-known/oauth-authorization-server" | jq . 2>/dev/null; then
    check_status "Authorization server metadata"
else
    echo -e "${RED}✗ Authorization server metadata failed${NC}"
fi

echo -e "\n${BLUE}Testing auth requirement (should return 401):${NC}"
HTTP_STATUS=$(curl -s -w "%{http_code}" -o /dev/null \
    -X POST "https://$APP_NAME.fly.dev/mcp" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}')

if [ "$HTTP_STATUS" = "401" ]; then
    echo -e "${GREEN}✓ Auth required (401 Unauthorized)${NC}"
else
    echo -e "${RED}✗ Expected 401, got $HTTP_STATUS${NC}"
fi

# Step 7: Instructions
echo -e "\n${CYAN}Step 7: Next Steps${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo -e "${GREEN}✓ Deployment complete!${NC}"
echo -e "\n${YELLOW}To register on Claude.ai:${NC}"
echo -e "  Name:        ${CYAN}Cowpilot Tools${NC} (no punctuation)"
echo -e "  Description: ${CYAN}MCP server providing various utility tools including echo, time, base64 encoding and more${NC}"
echo -e "  URL:         ${CYAN}https://$APP_NAME.fly.dev${NC}"

echo -e "\n${YELLOW}To monitor debug logs:${NC}"
echo -e "  ${BLUE}fly logs -a $APP_NAME${NC}"

echo -e "\n${YELLOW}To watch for registration attempts:${NC}"
echo -e "  ${BLUE}fly logs -a $APP_NAME | grep -E 'DEBUG|OAuth|CORS|http_request'${NC}"

echo -e "\n${YELLOW}To disable debug mode later:${NC}"
echo -e "  ${BLUE}fly secrets unset -a $APP_NAME MCP_DEBUG MCP_DEBUG_LEVEL MCP_DEBUG_STORAGE${NC}"

echo -e "\n${YELLOW}To check app health:${NC}"
echo -e "  ${BLUE}curl https://$APP_NAME.fly.dev/health${NC}"

echo -e "\n${GREEN}Happy debugging! 🎉${NC}"
