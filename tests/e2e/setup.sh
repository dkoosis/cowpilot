#!/bin/bash
# Set executable permissions for E2E test scripts

# Get the directory where this script is located
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Make all scripts in the same directory executable
chmod +x "$DIR"/*.sh

echo "Made all E2E test scripts executable:"
echo "  - mcp_scenarios.sh"
echo "  - verify-inspector.sh"
echo "  - manual-test-examples.sh"
echo "  - setup.sh"
echo "  - quick-setup.sh"
echo "  - raw_sse_test.sh"
echo "  - raw_examples.sh"
echo "  - RTFM_CORRECTION.md (documentation)"
