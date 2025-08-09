#!/bin/bash

# Quick diagnostic runner for OAuth issues

echo "RTM OAuth Diagnostic Suite"
echo "=========================="
echo ""

# Make scripts executable
chmod +x scripts/diagnostics/monitor_oauth.sh

# 1. Run basic monitoring
echo "1. Running OAuth endpoint monitoring..."
./scripts/diagnostics/monitor_oauth.sh $1

echo ""
echo "2. Running OAuth trace..."
go run ./scripts/diagnostics/oauth-trace $1

echo ""
echo "4. Checking local logs (if running locally)..."
if [ "$1" == "local" ]; then
    echo "Recent server output (last 50 lines):"
    # Check for any log files
    find . -name "*.log" -type f -exec tail -50 {} \; 2>/dev/null || echo "No log files found"
    
    # Check if server is running
    if pgrep -f "cmd/rtm/main.go" > /dev/null; then
        echo "✓ RTM server is running locally"
    else
        echo "✗ RTM server is not running. Start it with:"
        echo "  go run cmd/rtm/main.go"
    fi
else
    echo "Checking production logs..."
    if command -v flyctl &> /dev/null; then
        flyctl logs --app rtm --tail | head -100 | grep -E "\[OAuth" || echo "No OAuth logs found"
    else
        echo "Install flyctl to view production logs"
    fi
fi

echo ""
echo "Diagnostic Summary"
echo "------------------"
echo ""
echo "Common issues to check:"
echo "1. ✗ Missing WWW-Authenticate header → Auth middleware not working"
echo "2. ✗ 404 on discovery endpoints → Routes not registered"
echo "3. ✗ Callback not closing window → JavaScript completion signal failing"
echo "4. ✗ Token not found in store → Token storage/retrieval issue"
echo ""
echo "Next steps:"
echo "- Review the output above for any ✗ marks"
echo "- Check browser console during OAuth flow"
echo "- Monitor server logs in real-time during connection attempt"
