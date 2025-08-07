#!/bin/bash

# Make all diagnostic scripts executable
chmod +x scripts/diagnostics/*.sh

echo "OAuth Diagnostic Tools Ready"
echo "============================"
echo ""
echo "Available diagnostic tools:"
echo ""
echo "1. Quick diagnostic check:"
echo "   ./scripts/diagnostics/run_diagnostics.sh"
echo "   ./scripts/diagnostics/run_diagnostics.sh local  # for local testing"
echo ""
echo "2. OAuth endpoint monitor:"
echo "   ./scripts/diagnostics/monitor_oauth.sh"
echo ""
echo "3. Real-time server monitor (local only):"
echo "   go run scripts/diagnostics/monitor_realtime.go"
echo ""
echo "4. OAuth flow trace tool:"
echo "   go run scripts/diagnostics/oauth_trace.go"
echo "   go run scripts/diagnostics/oauth_trace.go local"
echo ""
echo "To diagnose your issue:"
echo "1. Run: ./scripts/diagnostics/run_diagnostics.sh"
echo "2. Try connecting in Claude while monitoring logs"
echo "3. Check browser console for errors"
echo "4. Look for any ✗ marks in the diagnostic output"
