# Troubleshooting Claude.ai Integration

## Common Issues

### CORS Errors
```
Access to fetch at 'https://mcp-adapters.fly.dev/mcp' from origin 'https://claude.ai' has been blocked by CORS policy
```
**Fix**: Ensure server is running with CORS middleware enabled. Check logs for "CORS: Enabled for [https://claude.ai]"

### SSE Connection Failed
```
Failed to establish SSE connection
```
**Fix**: 
- Verify server supports SSE: `curl -H "Accept: text/event-stream" https://mcp-adapters.fly.dev/mcp`
- Check fly.io deployment status: `fly status`

### Invalid Configuration
```
Connector configuration invalid
```
**Fix**:
- No punctuation in name (❌ "Claude.ai Tools" → ✅ "Claude ai Tools")
- Description must be >30 characters
- URL must be HTTPS

### Tools Not Appearing
**Fix**:
1. Test with inspector: `npx @modelcontextprotocol/inspector https://mcp-adapters.fly.dev/mcp --method tools/list`
2. Check server logs: `fly logs`
3. Verify all 11 tools are registered

## Debug Steps

1. **Test locally first**:
```bash
./bin/cowpilot  # In one terminal
npx @modelcontextprotocol/inspector http://localhost:8080/mcp --transport sse  # In another
```

2. **Check deployment**:
```bash
fly status
fly logs --tail
```

3. **Verify CORS headers**:
```bash
curl -I -X OPTIONS https://mcp-adapters.fly.dev/mcp \
  -H "Origin: https://claude.ai" \
  -H "Access-Control-Request-Method: POST"
```

## Known Limitations

- No OAuth yet (public access only)
- Stateless mode (no session persistence)
- Browser security prevents local server access
