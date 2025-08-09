#!/bin/bash

# RTM OAuth Diagnostic Script
# This script helps diagnose RTM OAuth integration issues

set -e

echo "==================================="
echo "RTM OAuth Integration Diagnostics"
echo "==================================="
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check environment variable
check_env() {
    local var_name=$1
    local var_value=${!var_name}
    
    if [ -z "$var_value" ]; then
        echo -e "${RED}✗${NC} $var_name is not set"
        return 1
    else
        # Mask sensitive values
        if [[ "$var_name" == *"SECRET"* ]] || [[ "$var_name" == *"TOKEN"* ]]; then
            echo -e "${GREEN}✓${NC} $var_name is set (${#var_value} characters)"
        else
            echo -e "${GREEN}✓${NC} $var_name = $var_value"
        fi
        return 0
    fi
}

# Function to test API endpoint
test_endpoint() {
    local url=$1
    local expected_status=$2
    local description=$3
    
    echo -n "Testing $description... "
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [ "$response" = "$expected_status" ]; then
        echo -e "${GREEN}✓${NC} (HTTP $response)"
        return 0
    else
        echo -e "${RED}✗${NC} (HTTP $response, expected $expected_status)"
        return 1
    fi
}

# Function to test RTM API directly
test_rtm_api() {
    local api_key=$1
    local api_secret=$2
    
    echo -n "Testing RTM API connectivity... "
    
    # Calculate signature for test call
    local method="rtm.test.echo"
    local params="api_key${api_key}methodrtm.test.echo"
    local sig=$(echo -n "${api_secret}${params}" | md5sum | cut -d' ' -f1)
    
    local url="https://api.rememberthemilk.com/services/rest/?method=rtm.test.echo&api_key=${api_key}&api_sig=${sig}&format=json"
    
    response=$(curl -s "$url" 2>/dev/null)
    
    if echo "$response" | grep -q '"stat":"ok"'; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        echo "  Response: $response"
        return 1
    fi
}

# Function to test OAuth flow
test_oauth_flow() {
    local server_url=$1
    
    echo ""
    echo "Testing OAuth endpoints:"
    echo "------------------------"
    
    # Test authorize endpoint
    test_endpoint "${server_url}/rtm/authorize?client_id=test&state=test&redirect_uri=http://localhost/callback" "200" "/rtm/authorize"
    
    # Test that token endpoint exists
    response=$(curl -s -X POST "${server_url}/rtm/token" -d "grant_type=authorization_code&code=invalid" 2>/dev/null)
    if echo "$response" | grep -q "invalid_grant"; then
        echo -e "${GREEN}✓${NC} /rtm/token endpoint responds correctly"
    else
        echo -e "${RED}✗${NC} /rtm/token endpoint issue"
    fi
}

# Function to run Go tests
run_tests() {
    echo ""
    echo "Running Go tests:"
    echo "-----------------"
    
    if command -v go &> /dev/null; then
        echo "Running unit tests..."
        if go test ./internal/rtm -v -count=1 2>&1 | grep -q "PASS"; then
            echo -e "${GREEN}✓${NC} Unit tests passed"
        else
            echo -e "${RED}✗${NC} Unit tests failed"
        fi
        
        echo "Running integration tests..."
        if go test ./internal/rtm -tags=integration -v -count=1 2>&1 | grep -q "PASS"; then
            echo -e "${GREEN}✓${NC} Integration tests passed"
        else
            echo -e "${YELLOW}⚠${NC} Integration tests failed or skipped"
        fi
    else
        echo -e "${YELLOW}⚠${NC} Go not installed, skipping tests"
    fi
}

# Main diagnostic flow
echo "1. Checking Environment"
echo "-----------------------"

# Determine if we're checking local or production
if [ "$1" = "--production" ] || [ "$1" = "-p" ]; then
    echo "Checking PRODUCTION (Fly.io) environment..."
    echo ""
    
    # Check Fly.io secrets
    if command -v fly &> /dev/null; then
        APP_NAME="${FLY_APP_NAME:-rtm-mcp}"
        if fly secrets list -a $APP_NAME 2>/dev/null | grep -q "RTM_API_KEY"; then
            echo -e "${GREEN}✓${NC} Fly.io secrets are configured"
            SERVER_URL="https://$APP_NAME.fly.dev"
            echo -e "${GREEN}✓${NC} Server URL: $SERVER_URL"
        else
            echo -e "${RED}✗${NC} Fly.io secrets not found for app: $APP_NAME"
            echo "  Run: fly secrets set RTM_API_KEY=... RTM_API_SECRET=... -a $APP_NAME"
            exit 1
        fi
    else
        echo -e "${RED}✗${NC} Fly CLI not installed"
        exit 1
    fi
else
    echo "Checking LOCAL environment..."
    echo "(Use --production flag to check Fly.io deployment)"
    echo ""
    
    all_env_good=true
    check_env "RTM_API_KEY" || all_env_good=false
    check_env "RTM_API_SECRET" || all_env_good=false
    check_env "SERVER_URL" || SERVER_URL="http://localhost:8080"
    check_env "PORT" || PORT=8080
    
    if [ "$all_env_good" = false ]; then
        echo ""
        echo -e "${YELLOW}⚠${NC} Missing local environment variables"
        echo ""
        echo "For LOCAL testing, set these:"
        echo "  export RTM_API_KEY=your_key"
        echo "  export RTM_API_SECRET=your_secret"
        echo "  export SERVER_URL=http://localhost:8080"
        echo ""
        echo "For PRODUCTION testing, run:"
        echo "  $0 --production"
        exit 1
    fi
fi

echo ""
echo "2. Testing RTM API Access"
echo "-------------------------"

if [ -n "$RTM_API_KEY" ] && [ -n "$RTM_API_SECRET" ]; then
    test_rtm_api "$RTM_API_KEY" "$RTM_API_SECRET"
else
    echo -e "${YELLOW}⚠${NC} Skipping RTM API test (credentials not available)"
fi

echo ""
echo "3. Testing Server Endpoints"
echo "----------------------------"

if [ -n "$SERVER_URL" ]; then
    # Check if server is running
    if curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL" 2>/dev/null | grep -q "000"; then
        echo -e "${RED}✗${NC} Server not reachable at $SERVER_URL"
        echo "  Make sure the server is running:"
        echo "  go run cmd/rtm/main.go"
    else
        test_oauth_flow "$SERVER_URL"
    fi
else
    echo -e "${YELLOW}⚠${NC} SERVER_URL not set, skipping endpoint tests"
fi

# Run Go tests
run_tests

echo ""
echo "4. Manual Test Commands"
echo "-----------------------"
echo ""
echo "Test frob generation:"
echo -e "${YELLOW}curl \"$SERVER_URL/rtm/test-frob\"${NC}"
echo ""
echo "Start OAuth flow:"
echo -e "${YELLOW}curl \"$SERVER_URL/rtm/authorize?client_id=test&state=xyz&redirect_uri=http://localhost:3000/callback\"${NC}"
echo ""
echo "Check auth status (after getting code):"
echo -e "${YELLOW}curl \"$SERVER_URL/rtm/check-auth?code=YOUR_CODE\"${NC}"
echo ""

echo "5. Debugging Checklist"
echo "----------------------"
echo ""
echo "If RTM registration is failing, check:"
echo "  □ RTM API credentials are valid (test at https://www.rememberthemilk.com/services/api/keys.rtm)"
echo "  □ Server is deployed to Fly.io (fly status)"
echo "  □ SERVER_URL uses HTTPS in production"
echo "  □ Cookies are not being blocked by browser"
echo "  □ User clicked 'OK, I'll allow it' on RTM page"
echo "  □ No timeout occurred (frobs expire after 60 minutes)"
echo "  □ Check server logs: fly logs"
echo ""

echo "6. Common Issues and Solutions"
echo "------------------------------"
echo ""
echo -e "${YELLOW}Issue:${NC} 'Authorization pending' forever"
echo -e "${GREEN}Fix:${NC} User needs to click 'OK, I'll allow it' on RTM page"
echo ""
echo -e "${YELLOW}Issue:${NC} 'Invalid CSRF token'"
echo -e "${GREEN}Fix:${NC} Check cookie settings, disable popup blockers"
echo ""
echo -e "${YELLOW}Issue:${NC} 'Invalid signature' (error 98)"
echo -e "${GREEN}Fix:${NC} Check RTM_API_SECRET is correct"
echo ""
echo -e "${YELLOW}Issue:${NC} 'Invalid frob' (error 101)"  
echo -e "${GREEN}Fix:${NC} Frob expired or user didn't authorize"
echo ""

echo "==================================="
echo "Diagnostic complete!"
echo "==================================="
