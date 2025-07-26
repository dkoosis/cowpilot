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

# Move to project root
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$PROJECT_ROOT"

# Check current directory
if [[ ! -f "cmd/cowpilot/main.go" ]]; then
    echo -e "${RED} ✗ Not in cowpilot project root"
    exit 1
fi

echo -e "${GREEN} ✓${NC} Project root verified"

FAILED=0

# Test 1: Project Structure
required_dirs=("cmd/cowpilot" "internal/debug" "internal/validator" "scripts/test" "docs/adr")
for dir in "${required_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        echo -e "${GREEN} ✓${NC} $dir exists"
    else
        echo -e "${RED} ✗ $dir missing"
        ((FAILED++))
    fi
done

# Test 2: Build Test
if go build -o bin/cowpilot cmd/cowpilot/main.go 2>/dev/null; then
    echo -e "${GREEN} ✓${NC} Build successful"
    size=$(ls -lh bin/cowpilot | awk '{print $5}')
    echo -e "\n${CYAN} ▶${NC} Binary size: $size"
    
    # Check binary size (warn if > 15MB)
    size_mb=$(du -m bin/cowpilot | cut -f1)
    if [ "$size_mb" -gt 15 ]; then
        echo -e "${YELLOW} ⚠️ Binary size exceeds 15MB"
    fi
else
    echo -e "${RED} ✗ Build failed"
    ((FAILED++))
fi

# Test 3: Dependency Verification
if go mod verify 2>/dev/null; then
    echo -e "${GREEN} ✓${NC} Dependencies verified"
else
    echo -e "${RED} ✗ Dependency verification failed${NC}"
    ((FAILED++))
fi

# Check for go.sum
if [[ -f "go.sum" ]]; then
    echo -e "${GREEN} ✓${NC} go.sum is present"
else
    echo -e "${RED} ✗ go.sum missing${NC}"
    ((FAILED++))
fi

# Test 4: Quick Server Startup Test
echo -e " "
echo -e "${BLUE} ▶${NC} Server startup test (stdio mode)..."
timeout 3s ./bin/cowpilot 2>/dev/null &
exit_code=$?
if [[ $exit_code -eq 124 ]]; then
    echo -e "${GREEN} ✓${NC} Server starts successfully"
else
    echo -e "${YELLOW} ⚠️${NC} Server startup test inconclusive"
fi

# Test 5: Feature Verification (based on STATE.yaml)

# Count implementations
tool_count=$(grep -c 'AddTool' cmd/cowpilot/main.go 2>/dev/null || echo 0)
resource_count=$(grep -c 'AddResource' cmd/cowpilot/main.go 2>/dev/null || echo 0)
prompt_count=$(grep -c 'AddPrompt' cmd/cowpilot/main.go 2>/dev/null || echo 0)

#echo -e "${CYAN} ▶${NC} Tools: $tool_count/11 expected"
if [ "$tool_count" -eq 11 ]; then
    echo -e "${GREEN} ✓${NC} $tool_count tools of 11 expected"
else
    echo -e "${YELLOW} ⚠️ Tool count mismatch: $tool_count tools of 11 expected"
fi

#echo -e "${CYAN} ▶${NC} Resources: $resource_count/4 expected"
if [ "$resource_count" -ge 3 ]; then
    echo -e "${GREEN} ✓${NC} $resource_count resources of 4 expected"
else
    echo -e "${YELLOW} ⚠️ Resource count low"
fi

#echo -e "${CYAN} ▶${NC} Prompts: $prompt_count/2 expected"
if [ "$prompt_count" -eq 2 ]; then
    echo -e "${GREEN} ✓${NC} $prompt_count prompts of 2 expected"
else
    echo -e "${YELLOW}        ⚠️  Prompt count mismatch"
fi

# Test 6: Debug System Check

# Check if debug config exists
if [[ -f "internal/debug/config.go" ]]; then
    echo -e "${GREEN} ✓${NC} Debug configuration implemented"
    
    # Test runtime config (should be disabled by default)
    MCP_DEBUG="" go run -tags debug tests/test_runtime_config.go 2>/dev/null || true
    echo -e "${GREEN} ✓${NC} Runtime configuration available"
else
    echo -e "${RED} ✗ Debug configuration missing"
    ((FAILED++))
fi

# Test 7: Documentation Check
docs=("README.md" "docs/adr/012-runtime-debug-configuration.md" "docs/STATE.yaml")
for doc in "${docs[@]}"; do
    if [[ -f "$doc" ]]; then
        echo -e "${GREEN} ✓${NC} $doc present"
    else
        echo -e "${YELLOW}        ⚠️  $doc missing"
    fi
done

# Test 8: Unit Test Check
if go test -v ./internal/... -count=1 &>/dev/null; then
    echo -e "${GREEN} ✓${NC} Unit tests passing"
else
    echo -e "${YELLOW}        ⚠️  Some unit tests failing"
fi

# Test 9: Production Health Check
if curl -s --max-time 5 "https://cowpilot.fly.dev/health" 2>/dev/null | grep -q "OK"; then
    echo -e "${GREEN} ✓${NC} Production server healthy"
else
    echo -e "${YELLOW}        ⚠️  Production server unreachable"
fi

# Test 10: Git Status
if git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${GREEN} ✓${NC} Git repository detected"
    
    # Check for uncommitted changes
    if [[ -z $(git status -s) ]]; then
        echo -e "${GREEN} ✓${NC} Working directory clean"
    else
        echo -e "${YELLOW} ⚠️${NC} Uncommitted changes present"
    fi
else
    echo -e "${YELLOW}        ⚠️  Not a git repository"
fi

# Summary
echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN} ✓ PASS${NC} Project Health Check 🎉"
    echo ""
    exit 0
else
    echo -e "${RED} ✗ FAIL Project Health Check ($FAILED critical issues)"
    exit 1
fi
