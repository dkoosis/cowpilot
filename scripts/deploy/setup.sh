#!/bin/bash
# Make all deployment scripts executable

cd "$(dirname "$0")"
chmod +x deploy-debug-to-fly.sh
chmod +x monitor-registration.sh
chmod +x check-status.sh

echo "✓ All deployment scripts are now executable"
echo ""
echo "Available scripts:"
echo "  ./deploy-debug-to-fly.sh  - Deploy to fly.io with debug enabled"
echo "  ./check-status.sh         - Quick status check of deployment"
echo "  ./monitor-registration.sh - Monitor Claude.ai registration attempts"
