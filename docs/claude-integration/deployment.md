# Claude.ai Integration - Deployment Guide

## Quick Start

1. Deploy to fly.io:
```bash
fly deploy
```

2. Test SSE transport:
```bash
npx @modelcontextprotocol/inspector https://cowpilot.fly.dev/mcp --transport sse --method initialize
```

3. Add to Claude.ai:
- Go to Settings → Connectors
- Click "Add More"
- Enter URL: `https://cowpilot.fly.dev/mcp`
- Name: "Cowpilot Tools" (no punctuation)
- Description: "Development tools including encoding, JSON formatting, and text operations" (>30 chars)

## Environment Variables

- `CORS_ALLOWED_ORIGINS`: Additional origins (comma-separated)
- `PORT`: Server port (default: 8080)
- `MCP_DEBUG`: Enable debug logging

## Testing CORS

```bash
# Test preflight
curl -X OPTIONS https://cowpilot.fly.dev/mcp \
  -H "Origin: https://claude.ai" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type" -v

# Test SSE
curl -X POST https://cowpilot.fly.dev/mcp \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

## OAuth (Phase 2)

Not implemented yet. Claude.ai requires:
- OAuth 2.0 with Dynamic Client Registration
- Callback URL: `https://claude.ai/api/mcp/auth_callback`
- Token refresh support
