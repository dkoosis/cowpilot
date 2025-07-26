#!/bin/bash
# Test Runner - List and run available test scripts

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Change to test directory
cd "$(dirname "$0")"

# Make all scripts executable
chmod +x *.sh 2>/dev/null

# --- FIXED: Use standard arrays for better compatibility ---
declare -a script_names=(
    "project-health-check.sh"
    "mcp-protocol-smoke-test.sh"
    "mcp-inspector-integration-test.sh"
    "mcp-transport-diagnostics.sh"
    "sse-transport-test.sh"
    "debug-tools-integration-test.sh"
)
declare -a script_descriptions=(
    "Comprehensive project validation and build verification"
    "Basic MCP protocol operations via direct HTTP/JSON-RPC"
    "Compatibility testing with official MCP Inspector tool"
    "HTTP/SSE transport auto-detection and client detection"
    "Server-Sent Events protocol verification for browser clients"
    "Debug system functionality with runtime configuration"
)

# Helper function to get a test's description
get_description() {
    local script_name_to_find="$1"
    for i in "${!script_names[@]}"; do
        if [[ "${script_names[$i]}" == "$script_name_to_find" ]]; then
            echo "${script_descriptions[$i]}"
            return
        fi
    done
    echo "No description available"
}
# --- END FIX ---

# Function to run a test
run_test() {
    local test_script="$1"
    echo -e "${CYAN} ▶${NC} $test_script"
    echo -e " "

    if [[ -x "$test_script" ]]; then
        ./"$test_script"
        return $?
    else
        echo -e "${RED}Error: $test_script is not executable"
        return 1
    fi
}

# Check command line arguments
if [[ $# -eq 0 ]]; then
    # No arguments - show menu
    echo -e "${CYAN}Available tests:"
    echo ""
    
    # List tests with descriptions
    i=1
    declare -a test_files
    for test_file in *.sh; do
        if [[ "$test_file" != "make-executable.sh" && "$test_file" != "run-tests.sh" ]]; then
            test_files[$i]="$test_file"
            # --- FIXED: Use the helper function to get the description ---
            desc=$(get_description "$test_file")
            printf "${GREEN}%2d)${NC} %-35s ${YELLOW}%s${NC}\n" "$i" "$test_file" "$desc"
            ((i++))
        fi
    done
    
    echo ""
    echo -e "${CYAN}Usage:"
    echo "  ./run-tests.sh <number>     # Run specific test by number"
    echo "  ./run-tests.sh <test-name>   # Run specific test by name"
    echo "  ./run-tests.sh all           # Run all tests"
    echo "  ./run-tests.sh quick         # Run quick smoke tests only"
    echo ""
    echo -e "${CYAN}Examples:"
    echo "  ./run-tests.sh 1"
    echo "  ./run-tests.sh project-health-check.sh"
    echo "  ./run-tests.sh all"
    echo ""
    
elif [[ "$1" == "all" ]]; then
    # Run all tests
    echo -e "${CYAN}Running all tests..."
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
    echo -e "${BLUE}═══════════════════════════════════════════════════════════"
    echo -e "${CYAN}Test Summary:"
    echo -e "Total tests: $total_tests"
    echo -e "Passed: ${GREEN}$passed_tests"
    echo -e "Failed: ${RED}$((total_tests - passed_tests))"
    
    if [[ $passed_tests -eq $total_tests ]]; then
        echo -e "${GREEN}All tests passed!"
        exit 0
    else
        echo -e "${RED}Some tests failed!"
        exit 1
    fi
    
elif [[ "$1" == "quick" ]]; then
    # Run quick tests only
    echo -e "${CYAN}🔥 smoke tests..."
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
        echo -e " ✨ Tests passed!\n"
        exit 0
    else
        echo -e "${RED}Quick tests failed!"
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
    
    echo -e "${RED}Error: Invalid test number"
    exit 1
    
else
    # Run test by name
    if [[ -f "$1" ]]; then
        run_test "$1"
        exit $?
    else
        echo -e "${RED}Error: Test '$1' not found"
        exit 1
    fi
fi