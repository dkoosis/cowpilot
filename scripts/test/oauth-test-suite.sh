#!/bin/bash
# OAuth Test Suite - Comprehensive OAuth implementation testing

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# gotestsum-style formatting
echo -e "🔐 OAuth Test Suite"
echo -e "${BLUE} ▶${NC} Running comprehensive OAuth tests..."

# Ensure we're in project root
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PROJECT_ROOT"

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Helper function to run test group
run_test_group() {
    local group_name="$1"
    local test_pattern="$2"
    local package="$3"
    
    echo -e "\n${CYAN}=== ${group_name} ===${NC}"
    
    # Run tests and capture output
    if output=$(go test -v "$package" -run "$test_pattern" 2>&1); then
        # Count individual test results
        local passed=$(echo "$output" | grep -c "PASS:" || true)
        local failed=$(echo "$output" | grep -c "FAIL:" || true)
        
        TOTAL_TESTS=$((TOTAL_TESTS + passed + failed))
        PASSED_TESTS=$((PASSED_TESTS + passed))
        FAILED_TESTS=$((FAILED_TESTS + failed))
        
        # Display individual test results
        echo "$output" | grep -E "RUN|PASS|FAIL" | while read line; do
            if echo "$line" | grep -q "RUN"; then
                test_name=$(echo "$line" | sed 's/.*RUN[[:space:]]*//')
                echo -e "${BLUE}  ${NC} ${test_name}..."
            elif echo "$line" | grep -q "PASS:"; then
                echo -e "${GREEN} ✓${NC} $(echo "$line" | sed 's/.*PASS:[[:space:]]*//' | cut -d' ' -f1)"
            elif echo "$line" | grep -q "FAIL:"; then
                echo -e "${RED} ✗${NC} $(echo "$line" | sed 's/.*FAIL:[[:space:]]*//' | cut -d' ' -f1)"
            fi
        done
        
        echo -e "${GREEN} ✓ Group passed${NC} (${passed} tests)"
        return 0
    else
        # Parse failures
        local failed=$(echo "$output" | grep -c "FAIL:" || true)
        TOTAL_TESTS=$((TOTAL_TESTS + failed))
        FAILED_TESTS=$((FAILED_TESTS + failed))
        
        echo "$output" | grep -E "FAIL:|Error:" | while read line; do
            echo -e "${RED} ✗${NC} $line"
        done
        
        echo -e "${RED} ✗ Group failed${NC} (${failed} tests)"
        return 1
    fi
}

# OAuth Adapter Tests
echo -e "\n${CYAN}--- OAuth Core Components ---${NC}"

echo -e "${BLUE} ▶${NC} Testing OAuth adapter..."
if run_test_group "OAuth Adapter" "TestOAuthAdapter" "./internal/auth"; then
    echo -e "${GREEN} ✓${NC} OAuth adapter tests passed"
else
    echo -e "${RED} ✗${NC} OAuth adapter tests failed"
fi

echo -e "\n${BLUE} ▶${NC} Testing CSRF protection..."
if run_test_group "CSRF Tokens" "TestCSRFTokens" "./internal/auth"; then
    echo -e "${GREEN} ✓${NC} CSRF protection verified"
else
    echo -e "${RED} ✗${NC} CSRF protection issues detected"
fi

echo -e "\n${BLUE} ▶${NC} Testing token storage..."
if run_test_group "Token Store" "TestTokenStore" "./internal/auth"; then
    echo -e "${GREEN} ✓${NC} Token storage working correctly"
else
    echo -e "${RED} ✗${NC} Token storage failures"
fi

echo -e "\n${BLUE} ▶${NC} Testing OAuth middleware..."
if run_test_group "OAuth Middleware" "TestOAuthMiddleware" "./internal/auth"; then
    echo -e "${GREEN} ✓${NC} Middleware authentication working"
else
    echo -e "${RED} ✗${NC} Middleware authentication issues"
fi

# Callback Server Tests
echo -e "\n${CYAN}--- OAuth Callback Server ---${NC}"

echo -e "${BLUE} ▶${NC} Testing callback server lifecycle..."
if run_test_group "Callback Server" "TestOAuthCallbackServer" "./internal/auth"; then
    echo -e "${GREEN} ✓${NC} Callback server operational"
else
    echo -e "${RED} ✗${NC} Callback server issues"
fi

# Integration Tests
echo -e "\n${CYAN}--- OAuth Integration Scenarios ---${NC}"

echo -e "${BLUE} ▶${NC} Testing complete OAuth flow..."
if run_test_group "OAuth Flow Integration" "TestOAuthFlow" "./tests"; then
    echo -e "${GREEN} ✓${NC} Complete OAuth flow working"
else
    echo -e "${RED} ✗${NC} OAuth flow integration failed"
fi

# Summary
echo -e "\n${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}OAuth Test Summary:${NC}"
echo -e "Total tests:  ${TOTAL_TESTS}"
echo -e "Passed:      ${GREEN}${PASSED_TESTS}${NC}"
echo -e "Failed:      ${RED}${FAILED_TESTS}${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN} ✨ All OAuth tests passed!${NC}"
    echo -e "${GREEN} ✓ PASS${NC} OAuth Test Suite"
    exit 0
else
    echo -e "\n${RED} ✗ Some OAuth tests failed${NC}"
    echo -e "${RED} ✗ FAIL${NC} OAuth Test Suite"
    exit 1
fi
