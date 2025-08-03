# OAuth Deployment Guide

## Environment Variables

```bash
# Required for production
SERVER_URL=https://mcp-adapters.fly.dev
PORT=8080
CORS_ALLOWED_ORIGINS=https://claude.ai

# Optional
MCP_DEBUG=true
```

## Testing OAuth Flow Locally

1. Start server:
```bash
./bin/cowpilot
```

2. Test OAuth endpoints:
```bash
./scripts/test/test-oauth-flow.sh
```

3. Complete auth flow:
- Visit: http://localhost:8080/oauth/authorize?client_id=test&redirect_uri=https://claude.ai/api/mcp/auth_callback
- Enter RTM API key
- Exchange code for token

## Claude.ai Setup

1. Deploy to fly.io:
```bash
fly deploy
fly env set SERVER_URL=https://mcp-adapters.fly.dev
```

2. Add to Claude.ai:
- Settings → Connectors → Add More
- URL: `https://mcp-adapters.fly.dev`
- Name: "mcp adapters RTM Tools"
- Description: "Remember The Milk integration with task management tools"

3. Connect:
- Click "Connect"
- Enter RTM API key when prompted
- Authorize access

## Troubleshooting

### 401 Unauthorized
- Check if token is being sent
- Verify auth endpoints are accessible
- Check CORS headers

### OAuth Flow Fails
- Ensure SERVER_URL is set correctly
- Check redirect_uri matches Claude's callback
- Verify all OAuth endpoints respond

### Token Invalid
- Tokens expire after 24 hours
- Re-authenticate through Claude.ai
