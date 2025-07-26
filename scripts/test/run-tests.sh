#!/bin/bash
# Test Runner - List and run available test scripts

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           ${CYAN}Cowpilot Test Suite Runner${BLUE}                    ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════╝${NC}"
echo ""

# Change to test directory
cd "$(dirname "$0")"

# Make all scripts executable
chmod +x *.sh 2>/dev/null

# Define test descriptions
declare -A test_descriptions=(
    ["project-health-check.sh"]="Comprehensive project validation and build verification"
    ["mcp-protocol-smoke-test.sh"]="Basic MCP protocol operations via direct HTTP/JSON-RPC"
    ["mcp-inspector-integration-test.sh"]="Compatibility testing with official MCP Inspector tool"
    ["mcp-transport-diagnostics.sh"]="HTTP/SSE transport auto-detection and client detection"
    ["sse-transport-test.sh"]="Server-Sent Events protocol verification for browser clients"
    ["debug-tools-integration-test.sh"]="Debug system functionality with runtime configuration"
)

# Function to run a test
run_test() {
    local test_script="$1"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}Running: $test_script${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    
    if [[ -x "$test_script" ]]; then
        ./"$test_script"
        return $?
    else
        echo -e "${RED}Error: $test_script is not executable${NC}"
        return 1
    fi
}

# Check command line arguments
if [[ $# -eq 0 ]]; then
    # No arguments - show menu
    echo -e "${CYAN}Available tests:${NC}"
    echo ""
    
    # List tests with descriptions
    i=1
    declare -a test_files
    for test_file in *.sh; do
        if [[ "$test_file" != "make-executable.sh" && "$test_file" != "run-tests.sh" ]]; then
            test_files[$i]="$test_file"
            desc="${test_descriptions[$test_file]:-No description available}"
            printf "${GREEN}%2d)${NC} %-35s ${YELLOW}%s${NC}\n" "$i" "$test_file" "$desc"
            ((i++))
        fi
    done
    
    echo ""
    echo -e "${CYAN}Usage:${NC}"
    echo "  ./run-tests.sh <number>     # Run specific test by number"
    echo "  ./run-tests.sh <test-name>   # Run specific test by name"
    echo "  ./run-tests.sh all           # Run all tests"
    echo "  ./run-tests.sh quick         # Run quick smoke tests only"
    echo ""
    echo -e "${CYAN}Examples:${NC}"
    echo "  ./run-tests.sh 1"
    echo "  ./run-tests.sh project-health-check.sh"
    echo "  ./run-tests.sh all"
    echo ""
    
elif [[ "$1" == "all" ]]; then
    # Run all tests
    echo -e "${CYAN}Running all tests...${NC}"
    echo ""
    
    total_tests=0
    passed_tests=0
    
    for test_file in *.sh; do
        if [[ "$test_file" != "make-executable.sh" && "$test_file" != "run-tests.sh" ]]; then
            ((total_tests++))
            if run_test "$test_file"; then
                ((passed_tests++))
            fi
            echo ""
        fi
    done
    
    # Summary
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}Test Summary:${NC}"
    echo -e "Total tests: $total_tests"
    echo -e "Passed: ${GREEN}$passed_tests${NC}"
    echo -e "Failed: ${RED}$((total_tests - passed_tests))${NC}"
    
    if [[ $passed_tests -eq $total_tests ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
    
elif [[ "$1" == "quick" ]]; then
    # Run quick tests only
    echo -e "${CYAN}Running quick smoke tests...${NC}"
    echo ""
    
    quick_tests=(
        "project-health-check.sh"
        "mcp-protocol-smoke-test.sh"
    )
    
    total_tests=0
    passed_tests=0
    
    for test_file in "${quick_tests[@]}"; do
        if [[ -f "$test_file" ]]; then
            ((total_tests++))
            if run_test "$test_file"; then
                ((passed_tests++))
            fi
            echo ""
        fi
    done
    
    if [[ $passed_tests -eq $total_tests ]]; then
        echo -e "${GREEN}Quick tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Quick tests failed!${NC}"
        exit 1
    fi
    
elif [[ "$1" =~ ^[0-9]+$ ]]; then
    # Run test by number
    i=1
    for test_file in *.sh; do
        if [[ "$test_file" != "make-executable.sh" && "$test_file" != "run-tests.sh" ]]; then
            if [[ $i -eq $1 ]]; then
                run_test "$test_file"
                exit $?
            fi
            ((i++))
        fi
    done
    
    echo -e "${RED}Error: Invalid test number${NC}"
    exit 1
    
else
    # Run test by name
    if [[ -f "$1" ]]; then
        run_test "$1"
        exit $?
    else
        echo -e "${RED}Error: Test '$1' not found${NC}"
        exit 1
    fi
fi
