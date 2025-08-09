#!/bin/bash

# Check Fly.io secrets for RTM app

echo "Checking Fly.io RTM Secrets"
echo "============================"
echo ""

# Check if fly CLI is installed
if ! command -v fly &> /dev/null; then
    echo "❌ Fly CLI not installed. Install from https://fly.io/docs/hands-on/install-flyctl/"
    exit 1
fi

# Check if logged in
if ! fly auth whoami &> /dev/null; then
    echo "❌ Not logged in to Fly. Run: fly auth login"
    exit 1
fi

APP_NAME="${FLY_APP_NAME:-rtm-mcp}"

echo "Checking secrets for app: $APP_NAME"
echo ""

# List secrets (won't show values, just names)
echo "Configured secrets:"
fly secrets list -a $APP_NAME 2>/dev/null | grep -E "RTM_|SERVER_URL|PORT" || {
    echo "❌ No RTM secrets found or app doesn't exist"
    echo ""
    echo "To set secrets, run:"
    echo "  fly secrets set RTM_API_KEY=your_key -a $APP_NAME"
    echo "  fly secrets set RTM_API_SECRET=your_secret -a $APP_NAME"
    echo "  fly secrets set SERVER_URL=https://$APP_NAME.fly.dev -a $APP_NAME"
    exit 1
}

echo ""
echo "✅ Secrets are configured in Fly.io"
echo ""
echo "To test the deployed app:"
echo "  curl https://$APP_NAME.fly.dev/health"
echo ""
echo "To view logs:"
echo "  fly logs -a $APP_NAME"
