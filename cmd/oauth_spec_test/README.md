# MCP OAuth Spec Compliance Test

This test determines if Claude.ai supports the **new MCP OAuth specification** (June 18, 2025) vs the old pattern.

## What This Tests

### ✅ New Spec Features (June 18, 2025):
- **OAuth 2.0 Protected Resource Metadata (RFC 9728)**: Resource server advertises auth server location
- **Resource Indicators (RFC 8707)**: Tokens scoped to specific resources  
- **Separation of Concerns**: Resource server separate from authorization server
- **Resource Server Pattern**: MCP server only validates tokens, doesn't issue them

### ❌ Old Pattern (March 2025):
- MCP server acts as both auth server AND resource server
- Direct OAuth endpoints on MCP server (`/oauth/authorize`, `/oauth/token`)
- Complex dual responsibility

## How to Run

```bash
# Make script executable
chmod +x run-test.sh

# Run the test
./run-test.sh
```

## Test Architecture

```
┌─────────────────┐    ┌──────────────────┐
│ Authorization   │    │ Resource Server  │
│ Server          │    │ (MCP Server)     │
│ :8091           │    │ :8090            │
│                 │    │                  │
│ /authorize      │    │ /mcp             │
│ /token          │    │ /.well-known/... │
└─────────────────┘    └──────────────────┘
```

## Testing with Claude.ai

1. **Start the test server**: `./run-test.sh`
2. **Add to Claude.ai**: Try to register `http://localhost:8090/mcp`
3. **Observe behavior**:
   - **New spec**: Claude discovers auth server, uses resource indicators
   - **Old spec**: Claude expects `/oauth/authorize` on MCP server directly

## Expected Results

### If Claude supports NEW spec:
```
📋 GET /.well-known/oauth-protected-resource
🔐 Claude redirects to auth server at :8091
🎫 Token request includes resource parameter
✅ MCP requests work with scoped token
```

### If Claude uses OLD spec:
```
❌ GET /oauth/authorize (404 - not found)
❌ No metadata discovery
❌ Direct OAuth endpoints expected on MCP server
```

## Interpreting Results

| Behavior | Claude Spec Support |
|----------|-------------------|
| Fetches `/.well-known/oauth-protected-resource` | ✅ New spec |
| Redirects to separate auth server | ✅ New spec |  
| Uses resource indicators | ✅ New spec |
| Expects `/oauth/authorize` on MCP server | ❌ Old spec |
| No metadata discovery | ❌ Old spec |

## Files

- `main.go`: Test server implementation
- `run-test.sh`: Test runner script  
- `README.md`: This documentation

## Decision Impact

- **New spec supported** → Migrate cowpilot to resource server pattern
- **Old spec only** → Fix current auth UX issues first, migrate later
