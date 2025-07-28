#!/bin/bash
# Quick deploy and monitor script for mcp adapters

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Make sure scripts are executable
chmod +x "$SCRIPT_DIR"/*.sh

# Deploy
echo "Starting deployment..."
"$SCRIPT_DIR/deploy-debug-to-fly.sh"

# If deployment succeeded, start monitoring
if [ $? -eq 0 ]; then
    echo ""
    echo "Deployment successful! Starting monitor..."
    sleep 2
    "$SCRIPT_DIR/monitor-registration.sh"
else
    echo "Deployment failed. Check the errors above."
    exit 1
fi
