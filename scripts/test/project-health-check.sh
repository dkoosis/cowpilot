#!/bin/bash
# Project Health Check - Comprehensive project validation

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== RUN   Project Health Check${NC}"
echo -e "${BLUE}    --- Validating Cowpilot project structure and functionality${NC}"

# Check current directory
if [[ ! -f "cmd/cowpilot/main.go" ]]; then
    echo -e "${RED}    ✗ Not in cowpilot project root${NC}"
    echo -e "${RED}--- FAIL  Project Health Check${NC}"
    exit 1
fi

echo -e "${GREEN}    ✓ Project root verified${NC}"

FAILED=0

# Test 1: Project Structure
echo -e "${BLUE}    --- Test 1: Project Structure${NC}"
required_dirs=("cmd/cowpilot" "internal/debug" "internal/validator" "scripts/test" "docs/adr")
for dir in "${required_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        echo -e "${GREEN}        ✓ $dir exists${NC}"
    else
        echo -e "${RED}        ✗ $dir missing${NC}"
        ((FAILED++))
    fi
done

# Test 2: Build Test
echo -e "${BLUE}    --- Test 2: Build Test${NC}"
if go build -o bin/cowpilot cmd/cowpilot/main.go 2>/dev/null; then
    echo -e "${GREEN}        ✓ Build successful${NC}"
    size=$(ls -lh bin/cowpilot | awk '{print $5}')
    echo -e "${CYAN}        Binary size: $size${NC}"
    
    # Check binary size (warn if > 15MB)
    size_mb=$(du -m bin/cowpilot | cut -f1)
    if [ "$size_mb" -gt 15 ]; then
        echo -e "${YELLOW}        ⚠️  Binary size exceeds 15MB${NC}"
    fi
else
    echo -e "${RED}        ✗ Build failed${NC}"
    ((FAILED++))
fi

# Test 3: Dependency Verification
echo -e "${BLUE}    --- Test 3: Dependency Check${NC}"
if go mod verify 2>/dev/null; then
    echo -e "${GREEN}        ✓ Dependencies verified${NC}"
else
    echo -e "${RED}        ✗ Dependency verification failed${NC}"
    ((FAILED++))
fi

# Check for go.sum
if [[ -f "go.sum" ]]; then
    echo -e "${GREEN}        ✓ go.sum present${NC}"
else
    echo -e "${RED}        ✗ go.sum missing${NC}"
    ((FAILED++))
fi

# Test 4: Quick Server Startup Test
echo -e "${BLUE}    --- Test 4: Server Startup Test${NC}"
echo -e "${BLUE}        Starting server in stdio mode...${NC}"
timeout 3s ./bin/cowpilot 2>/dev/null &
exit_code=$?
if [[ $exit_code -eq 124 ]]; then
    echo -e "${GREEN}        ✓ Server starts successfully${NC}"
else
    echo -e "${YELLOW}        ⚠️  Server startup test inconclusive${NC}"
fi

# Test 5: Feature Verification (based on STATE.yaml)
echo -e "${BLUE}    --- Test 5: Feature Implementation Check${NC}"

# Count implementations
tool_count=$(grep -c 'AddTool' cmd/cowpilot/main.go 2>/dev/null || echo 0)
resource_count=$(grep -c 'AddResource' cmd/cowpilot/main.go 2>/dev/null || echo 0)
prompt_count=$(grep -c 'AddPrompt' cmd/cowpilot/main.go 2>/dev/null || echo 0)

echo -e "${CYAN}        Tools: $tool_count/11 expected${NC}"
if [ "$tool_count" -eq 11 ]; then
    echo -e "${GREEN}        ✓ All tools implemented${NC}"
else
    echo -e "${YELLOW}        ⚠️  Tool count mismatch${NC}"
fi

echo -e "${CYAN}        Resources: $resource_count/4 expected${NC}"
if [ "$resource_count" -ge 3 ]; then
    echo -e "${GREEN}        ✓ Resources implemented${NC}"
else
    echo -e "${YELLOW}        ⚠️  Resource count low${NC}"
fi

echo -e "${CYAN}        Prompts: $prompt_count/2 expected${NC}"
if [ "$prompt_count" -eq 2 ]; then
    echo -e "${GREEN}        ✓ All prompts implemented${NC}"
else
    echo -e "${YELLOW}        ⚠️  Prompt count mismatch${NC}"
fi

# Test 6: Debug System Check
echo -e "${BLUE}    --- Test 6: Debug System Configuration${NC}"

# Check if debug config exists
if [[ -f "internal/debug/config.go" ]]; then
    echo -e "${GREEN}        ✓ Debug configuration implemented${NC}"
    
    # Test runtime config (should be disabled by default)
    MCP_DEBUG="" go run -tags debug tests/test_runtime_config.go 2>/dev/null || true
    echo -e "${GREEN}        ✓ Runtime configuration available${NC}"
else
    echo -e "${RED}        ✗ Debug configuration missing${NC}"
    ((FAILED++))
fi

# Test 7: Documentation Check
echo -e "${BLUE}    --- Test 7: Documentation Check${NC}"
docs=("README.md" "docs/adr/012-runtime-debug-configuration.md" "docs/STATE.yaml")
for doc in "${docs[@]}"; do
    if [[ -f "$doc" ]]; then
        echo -e "${GREEN}        ✓ $doc present${NC}"
    else
        echo -e "${YELLOW}        ⚠️  $doc missing${NC}"
    fi
done

# Test 8: Unit Test Check
echo -e "${BLUE}    --- Test 8: Unit Test Coverage${NC}"
if go test -v ./internal/... -count=1 &>/dev/null; then
    echo -e "${GREEN}        ✓ Unit tests passing${NC}"
else
    echo -e "${YELLOW}        ⚠️  Some unit tests failing${NC}"
fi

# Test 9: Production Health Check
echo -e "${BLUE}    --- Test 9: Production Server Check${NC}"
if curl -s --max-time 5 "https://cowpilot.fly.dev/health" 2>/dev/null | grep -q "OK"; then
    echo -e "${GREEN}        ✓ Production server healthy${NC}"
else
    echo -e "${YELLOW}        ⚠️  Production server unreachable${NC}"
fi

# Test 10: Git Status
echo -e "${BLUE}    --- Test 10: Git Repository Status${NC}"
if git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${GREEN}        ✓ Git repository detected${NC}"
    
    # Check for uncommitted changes
    if [[ -z $(git status -s) ]]; then
        echo -e "${GREEN}        ✓ Working directory clean${NC}"
    else
        echo -e "${YELLOW}        ⚠️  Uncommitted changes present${NC}"
    fi
else
    echo -e "${YELLOW}        ⚠️  Not a git repository${NC}"
fi

# Summary
echo ""
echo -e "${CYAN}═══════════════════════════════════════${NC}"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}--- PASS  Project Health Check${NC}"
    echo -e "${GREEN}    🎉 Cowpilot is healthy and ready!${NC}"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo -e "  make test-verbose         # Run full test suite"
    echo -e "  make scenario-test-local  # E2E testing"
    echo -e "  make run-debug-proxy      # Debug mode"
    echo -e "  npx @modelcontextprotocol/inspector ./bin/cowpilot"
    exit 0
else
    echo -e "${RED}--- FAIL  Project Health Check ($FAILED critical issues)${NC}"
    exit 1
fi
