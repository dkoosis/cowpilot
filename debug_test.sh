#!/bin/bash
# Debug version to see what's happening

set -x  # Enable debug output

SERVER_URL="${1:-https://cowpilot.fly.dev/}"
echo "SERVER_URL: $SERVER_URL"

HEALTH_URL="${SERVER_URL%/}/health"
echo "HEALTH_URL: $HEALTH_URL"

# Try the health check
curl -s -f "$HEALTH_URL"
echo "Curl exit code: $?"
