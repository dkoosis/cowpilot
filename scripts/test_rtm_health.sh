#!/bin/bash

# Test the RTM production health check

echo "Running RTM Production Health Test"
echo "=================================="
echo ""

# Run the Go test with nice formatting
if command -v gotestsum &> /dev/null; then
    GO_TEST=1 gotestsum --format testdox -- -v -timeout 30s ./tests/integration -run TestRTMProductionHealth
else
    GO_TEST=1 go test -v -timeout 30s ./tests/integration -run TestRTMProductionHealth
fi

exit_code=$?

echo ""
if [ $exit_code -eq 0 ]; then
    echo "✅ RTM Production Health: PASSED"
    echo ""
    echo "Ready to connect Claude Desktop to: https://rtm.fly.dev/mcp"
else
    echo "❌ RTM Production Health: FAILED"
    echo ""
    echo "Fix issues and run: make deploy-rtm"
fi

exit $exit_code
